package main

import (
	"encoding/json"
	"fmt"
	"github.com/kumahq/ci-tools/cmd/internal/github"
	"github.com/kumahq/ci-tools/cmd/internal/version"
	"github.com/kumahq/ci-tools/cmd/internal/versionfile"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"sort"
	"time"
)

var lifetimeMonths int
var edition string
var minVersion string
var includeDev bool
var activeBranches bool

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
		minMajor, minMinor, _ := version.MustSplitSemVer(minVersion)
		byVersion := map[string][]github.GQLRelease{}
		for i := range res {
			major, minor, _ := version.MustSplitSemVer(res[i].Name)
			if major < minMajor || (major == minMajor && minor < minMinor) {
				continue
			}
			release := fmt.Sprintf("%d.%d.x", major, minor)
			byVersion[release] = append(byVersion[release], res[i])
		}
		var out []versionfile.VersionEntry
		for releaseName, releases := range byVersion {
			res, err := versionfile.BuildVersionEntry(edition, releaseName, lifetimeMonths, releases)
			if err != nil {
				return err
			}
			out = append(out, res)
		}
		sort.Slice(out, func(i, j int) bool {
			return out[i].Less(out[j])
		})
		if includeDev {
			out = append(out, versionfile.Dev(edition))
		}
		if activeBranches {
			var branches []string
			for _, v := range out {
				t, _ := time.Parse(time.DateOnly, v.EndOfLifeDate)
				if v.EndOfLifeDate == "" || time.Now().Before(t) {
					branches = append(branches, v.Branch)
				}
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(branches)
		}
		return yaml.NewEncoder(cmd.OutOrStdout()).Encode(out)
	},
}

func init() {
	versionFile.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
	versionFile.Flags().StringVar(&edition, "edition", "kuma", "The edition of the product")
	versionFile.Flags().IntVar(&lifetimeMonths, "lifetime-months", 12, "the number of months a version is valid for")
	versionFile.Flags().StringVar(&minVersion, "min-version", "1.2.0", "The minimum version to build a version files on")
	versionFile.Flags().BoolVar(&includeDev, "include-dev", true, "Skip dev")
	versionFile.Flags().BoolVar(&activeBranches, "active-branches", false, "only output a json with the branches not EOL")
}
