package main

import (
	"encoding/json"
	"fmt"
	"regexp"
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
		gqlClient := github.GqlClientFromEnv()
		res, err := gqlClient.ReleaseGraphQL(config.repo)
		if err != nil {
			return err
		}
		minVersionVer := semver.MustParse(minVersion)
		byVersion := map[string][]github.GQLRelease{}
		for i := range res {
			curVersion := res[i].SemVer()
			if curVersion.Prerelease() != "" {
				continue // Ignore prereleases
			}
			if curVersion.LessThan(minVersionVer) {
				continue
			}
			release := fmt.Sprintf("%d.%d.x", curVersion.Major(), curVersion.Minor())
			byVersion[release] = append(byVersion[release], res[i])
		}
		var out []versionfile.VersionEntry
		for releaseName, releases := range byVersion {
			res, err := versionfile.BuildVersionEntry(edition, releaseName, lifetimeMonths, ltsLifetimeMonths, releases)
			if err != nil {
				return err
			}
			out = append(out, res)
		}
		sort.Slice(out, func(i, j int) bool {
			return out[i].Less(out[j])
		})
		// Add the dev version
		devVersion := versionfile.VersionEntry{
			Edition: edition,
			Version: "preview",
			Branch:  "master",
			Label:   "dev",
			Release: regexp.MustCompile(`\.[0-9]+$`).ReplaceAllString(semver.MustParse(out[len(out)-1].Version).IncMinor().String(), ".x"),
		}
		out = append(out, devVersion)
		if activeBranches {
			var branches []string
			for _, v := range out {
				t, _ := time.Parse(time.DateOnly, v.EndOfLifeDate)
				if v.EndOfLifeDate == "" || time.Now().Before(t) {
					branches = append(branches, v.Branch)
				}
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(ActiveBranches{branches})
		}
		return yaml.NewEncoder(cmd.OutOrStdout()).Encode(out)
	},
}

func init() {
	versionFile.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
	versionFile.Flags().StringVar(&edition, "edition", "kuma", "The edition of the product")
	versionFile.Flags().IntVar(&lifetimeMonths, "lifetime-months", 12, "the number of months a version is valid for")
	versionFile.Flags().IntVar(&ltsLifetimeMonths, "lts-lifetime-months", 30, "the number of months an lts version is valid for")
	versionFile.Flags().StringVar(&minVersion, "min-version", "1.2.0", "The minimum version to build a version files on")
	versionFile.Flags().BoolVar(&activeBranches, "active-branches", false, "only output a json with the branches not EOL")
}
