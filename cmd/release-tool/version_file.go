package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/kumahq/ci-tools/cmd/internal/github"
	"github.com/kumahq/ci-tools/cmd/internal/versionfile"
)

var (
	lifetimeMonths    int
	ltsLifetimeMonths int
	edition           string
	minVersion        string
	activeBranches    bool
)

type ActiveBranches struct {
	BaseBranchPatterns []string `json:"baseBranchPatterns"`
}

var versionFile = &cobra.Command{
	Use:   "version-file",
	Short: "Recreate the versions.yaml using github releases",
	Long: `
	We use metadata from github to generate the versions file
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		gqlClient, err := github.NewGQLClient(config.useGHAuth)
		if err != nil {
			return err
		}

		releases, err := gqlClient.ReleaseGraphQL(config.repo)
		if err != nil {
			return err
		}

		out, branches, err := assembleVersions(
			edition, minVersion, lifetimeMonths, ltsLifetimeMonths,
			releases, releaseBranchNames(cmd.Context(), gqlClient, config.repo),
		)
		if err != nil {
			return err
		}

		if activeBranches {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(ActiveBranches{branches})
		}
		return yaml.NewEncoder(cmd.OutOrStdout()).Encode(out)
	},
}

// assembleVersions builds the versions.yml entries and the active branch list
// from fetched releases and branch names. A line whose releases are all drafts
// is skipped: the preview entry covers that line while the release is being
// prepared.
func assembleVersions(
	edition, minVersion string, lifetimeMonths, ltsLifetimeMonths int,
	releases []github.GQLRelease, branchNames []string,
) ([]versionfile.VersionEntry, []string, error) {
	minVersionVer := semver.MustParse(minVersion)
	byVersion := map[string][]github.GQLRelease{}
	for i := range releases {
		curVersion := releases[i].SemVer()
		if curVersion.Prerelease() != "" {
			continue
		}
		if curVersion.LessThan(minVersionVer) {
			continue
		}
		release := fmt.Sprintf("%d.%d.x", curVersion.Major(), curVersion.Minor())
		byVersion[release] = append(byVersion[release], releases[i])
	}
	var out []versionfile.VersionEntry
	for releaseName, lineReleases := range byVersion {
		if !slices.ContainsFunc(lineReleases, github.GQLRelease.IsReleased) {
			continue
		}
		entry, err := versionfile.BuildVersionEntry(edition, releaseName, lifetimeMonths, ltsLifetimeMonths, lineReleases)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, entry)
	}
	if len(out) == 0 {
		return nil, nil, fmt.Errorf("no published releases found in %d releases", len(releases))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Less(out[j])
	})
	latestReleased := latestReleasedVersion(releases)
	unreleased := newestUnreleasedBranch(latestReleased, branchNames)
	devRelease := regexp.MustCompile(`\.[0-9]+$`).ReplaceAllString(latestReleased.IncMinor().String(), ".x")
	if unreleased != nil {
		devRelease = branchPattern(unreleased)
	}
	out = append(out, versionfile.VersionEntry{
		Edition: edition,
		Version: "preview",
		Branch:  "master",
		Label:   "dev",
		Release: devRelease,
	})
	var branches []string
	for _, v := range out {
		t, _ := time.Parse(time.DateOnly, v.EndOfLifeDate)
		if v.EndOfLifeDate == "" || time.Now().Before(t) {
			branches = append(branches, v.Branch)
		}
	}
	if unreleased != nil {
		branches = insertBeforeDefault(branches, branchName(unreleased))
	}
	return out, branches, nil
}

var releaseBranchRe = regexp.MustCompile(`^release-(\d+)\.(\d+)$`)

// releaseBranchNames fetches the repo's branch names for unreleased-branch
// detection. The lookup is advisory: on failure it returns nil so the daily
// regeneration falls back to the release-only output instead of failing.
func releaseBranchNames(ctx context.Context, cl *github.GQLClient, repo string) []string {
	names, err := cl.ReleaseBranches(ctx, repo)
	if err != nil {
		return nil
	}
	return names
}

// latestReleasedVersion returns the highest version among published
// (non-draft, non-prerelease) releases. Drafts are excluded so a release
// being prepared does not mask its own branch.
func latestReleasedVersion(releases []github.GQLRelease) *semver.Version {
	var latest *semver.Version
	for i := range releases {
		if !releases[i].IsReleased() {
			continue
		}
		v := releases[i].SemVer()
		if latest == nil || v.GreaterThan(latest) {
			latest = v
		}
	}
	return latest
}

// newestUnreleasedBranch returns the version of the newest release-X.Y branch
// that is ahead of every published release line, or nil when no branch is
// ahead (the normal state right after a release).
func newestUnreleasedBranch(latestReleased *semver.Version, branches []string) *semver.Version {
	var newest *semver.Version
	for _, b := range branches {
		m := releaseBranchRe.FindStringSubmatch(b)
		if m == nil {
			continue
		}
		v := semver.MustParse(m[1] + "." + m[2] + ".0")
		if latestReleased != nil && v.Compare(latestReleased) <= 0 {
			continue
		}
		if newest == nil || v.GreaterThan(newest) {
			newest = v
		}
	}
	return newest
}

func branchPattern(v *semver.Version) string {
	return fmt.Sprintf("%d.%d.x", v.Major(), v.Minor())
}

func branchName(v *semver.Version) string {
	return fmt.Sprintf("release-%d.%d", v.Major(), v.Minor())
}

// insertBeforeDefault puts an unreleased branch before the default branch
// ("master"), which is always last, so consumers see it during the release cycle.
func insertBeforeDefault(branches []string, unreleased string) []string {
	if slices.Contains(branches, unreleased) {
		return branches
	}
	out := make([]string, 0, len(branches)+1)
	inserted := false
	for _, b := range branches {
		if !inserted && b == "master" {
			out = append(out, unreleased)
			inserted = true
		}
		out = append(out, b)
	}
	if !inserted {
		out = append(out, unreleased)
	}
	return out
}

func init() {
	versionFile.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
	versionFile.Flags().StringVar(&edition, "edition", "kuma", "The edition of the product")
	versionFile.Flags().IntVar(&lifetimeMonths, "lifetime-months", 12, "the number of months a version is valid for")
	versionFile.Flags().IntVar(&ltsLifetimeMonths, "lts-lifetime-months", 24, "the number of months an lts version is valid for")
	versionFile.Flags().StringVar(&minVersion, "min-version", "1.2.0", "The minimum version to build a version files on")
	versionFile.Flags().BoolVar(&activeBranches, "active-branches", false, "only output a json with the branches not EOL")
}
