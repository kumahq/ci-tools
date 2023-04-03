package main

import (
	"encoding/json"
	"fmt"
	"github.com/kumahq/ci-tools/cmd/internal/github"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var lifetimeMonths int
var edition string
var minVersion string
var includeDev bool
var activeBranches bool

type VersionEntry struct {
	Edition       string `yaml:"edition"`
	Version       string `yaml:"version"`
	Release       string `yaml:"release"`
	Latest        bool   `yaml:"latest,omitempty"`
	ReleaseDate   string `yaml:"releaseDate"`
	EndOfLifeDate string `yaml:"endOfLifeDate"`
	Branch        string `yaml:"branch"`
}

var releasedOnRegexp = regexp.MustCompile("^> Released on ([0-9]{4}/[0-9]{2}/[0-9]{2})")

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
		minMajor, minMinor, _ := MustSplitSemVer(minVersion)
		byVersion := map[string][]github.GQLRelease{}
		for i := range res {
			major, minor, _ := MustSplitSemVer(res[i].Name)
			if major < minMajor || (major == minMajor && minor < minMinor) {
				continue
			}
			release := fmt.Sprintf("%d.%d.x", major, minor)
			byVersion[release] = append(byVersion[release], res[i])
		}
		var out []VersionEntry
		for v, releases := range byVersion {
			latest := false
			for _, r := range releases {
				latest = latest || r.IsLatest
			}
			sort.Slice(releases, func(i, j int) bool {
				iv, _ := strconv.Atoi(strings.Split(releases[i].Name, ".")[2])
				jv, _ := strconv.Atoi(strings.Split(releases[j].Name, ".")[2])
				return iv < jv
			})
			releaseDate, err := ExtractReleaseDate(releases[0])
			if err != nil {
				return fmt.Errorf("failed to extract release date for %s because of: %s", releases[0].Name, err.Error())
			}
			EOLDate := releaseDate.AddDate(0, lifetimeMonths, 0)
			entry := VersionEntry{
				Release:       v,
				Edition:       edition,
				Version:       releases[len(releases)-1].Name,
				Latest:        latest,
				ReleaseDate:   releaseDate.Format(time.DateOnly),
				EndOfLifeDate: EOLDate.Format(time.DateOnly),
				Branch:        ExtractBranch(releases[0].Name),
			}
			out = append(out, entry)
		}
		sort.Slice(out, func(i, j int) bool {
			majorI, minorI, _ := MustSplitSemVer(strings.ReplaceAll(out[i].Release, "x", "0"))
			majorJ, minorJ, _ := MustSplitSemVer(strings.ReplaceAll(out[j].Release, "x", "0"))
			if majorI == majorJ {
				return minorI < minorJ
			}
			return majorI < majorJ
		})
		if includeDev {
			out = append(out, VersionEntry{
				Release:       "dev",
				Edition:       edition,
				Version:       "preview",
				Branch:        "master",
				ReleaseDate:   "2020-01-01",
				EndOfLifeDate: "2030-01-01",
			})
		}
		if activeBranches {
			var branches []string
			for _, v := range out {
				t, _ := time.Parse(time.DateOnly, v.EndOfLifeDate)
				if time.Now().Before(t) {
					branches = append(branches, v.Branch)
				}
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(branches)
		}
		return yaml.NewEncoder(cmd.OutOrStdout()).Encode(out)
	},
}

func ExtractBranch(name string) string {
	major, minor, _ := MustSplitSemVer(name)
	return fmt.Sprintf("release-%d.%d", major, minor)
}

func ExtractReleaseDate(r github.GQLRelease) (time.Time, error) {
	res := releasedOnRegexp.FindStringSubmatch(r.Description)
	if len(res) == 2 {
		return time.Parse("2006/01/02", res[1])
	}
	return r.PublishedAt, nil
}

func init() {
	versionFile.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
	versionFile.Flags().StringVar(&edition, "edition", "kuma", "The edition of the product")
	versionFile.Flags().IntVar(&lifetimeMonths, "lifetime-months", 12, "the number of months a version is valid for")
	versionFile.Flags().StringVar(&minVersion, "min-version", "1.2.0", "The minimum version to build a version files on")
	versionFile.Flags().BoolVar(&includeDev, "include-dev", true, "Skip dev")
	versionFile.Flags().BoolVar(&activeBranches, "active-branches", false, "only output a json with the branches not EOL")
}
