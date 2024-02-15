package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kumahq/ci-tools/cmd/internal/changeloggenerator"
	"github.com/kumahq/ci-tools/cmd/internal/github"
)

type OutFormat string

const (
	FormatMarkdown OutFormat = "md"
	FormatJson     OutFormat = "json"
)

var autoChangelog = &cobra.Command{
	Use:   "changelog.md",
	Short: "Recreate the changelog.md using the changelog in each github release",
	Long: `
	We use whatever is after '## Changelog' to build the changelog
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		gqlClient := github.GqlClientFromEnv()
		res, err := gqlClient.ReleaseGraphQL(config.repo)
		if err != nil {
			return err
		}
		sort.SliceStable(res, func(i, j int) bool {
			if res[i].IsLatest {
				return true
			}
			if res[j].IsLatest {
				return false
			}
			// If they are release roughly at the same time we should sort them by semver order
			if res[i].CreatedAt.Truncate(time.Hour*24) == res[j].CreatedAt.Truncate(time.Hour*24) {
				return !res[i].SemVer().LessThan(res[j].SemVer())
			}
			return res[i].CreatedAt.After(res[j].CreatedAt)
		})
		_, _ = cmd.OutOrStdout().Write([]byte("# Changelog\n<!-- Autogenerated with (github.com/kumahq/ci-tools) release-tool changelog.md -->\n"))
		for _, release := range res {
			if !release.IsReleased() { // If the release is not an actual release don't add in changelog.md
				continue
			}
			if strings.Contains(release.Description, "## Changelog") {
				changelog := strings.SplitN(release.Description, "## Changelog", 2)[1]
				_, _ = cmd.OutOrStdout().Write([]byte(fmt.Sprintf(`
## %s
> Released on %s%s
`, release.Name, release.CreatedAt.Format("2006/01/02"), changelog)))
			}

		}
		return nil
	},
}

var versionChangelog = &cobra.Command{
	Use:   "version-changelog",
	Short: "Generate the changelog for a specific release using the github graphql api",
	Long: `Generate the changelog using the github graphql api.
This will get all the commits in the branch after '--from-tag'
It will retrieve all the associated PRs to these commits and extract a changelog entry following these rules:

- If there's in the PR description an entry '> Changelog:'
	- If it's 'skip' --> This PR won't be listed in the changelog
	- Use this as the value for the changelog
- If the PR title starts with ci, test, refactor, build... skip the entry (if you still want it add a '> Changelog:' line in the PR description.
- Else use the PR title in the changelog

It will then output a changelog with all PRs with the same changelog grouped together
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if config.fromTag == "" {
			return errors.New("You must set either --from-tag")
		}
		gqlClient := github.GqlClientFromEnv()
		out, err := getChangelog(gqlClient, config.repo, config.branch, config.fromTag)
		if err != nil {
			return err
		}
		switch OutFormat(config.format) {
		case FormatMarkdown:
			for _, v := range out {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "* %s\n", v)
				if err != nil {
					return err
				}
			}
		case FormatJson:
			e := json.NewEncoder(cmd.OutOrStdout())
			e.SetIndent("", "  ")
			return e.Encode(out)
		}
		return nil
	},
}

func getChangelog(gqlClient *github.GQLClient, repo string, branch string, tag string) (changeloggenerator.Changelog, error) {
	// Retrieve data from github
	commit, err := gqlClient.CommitByRef(repo, tag)
	if err != nil {
		return nil, err
	}
	// Deal with pagination
	res, err := gqlClient.HistoryGraphQl(repo, branch, commit)
	if err != nil {
		return nil, err
	}
	var commitInfos []changeloggenerator.CommitInfo
	for _, commit := range res {
		if len(commit.AssociatedPullRequests.Nodes) == 0 {
			continue
		}
		pr := commit.AssociatedPullRequests.Nodes[0]
		ci := changeloggenerator.CommitInfo{
			Author:        pr.Author.Login,
			Sha:           commit.Oid,
			PrNumber:      pr.Number,
			PrTitle:       pr.Title,
			PrBody:        pr.Body,
			CommitMessage: commit.Message,
		}
		commitInfos = append(commitInfos, ci)
	}
	return changeloggenerator.New(config.repo, commitInfos)
}

func init() {
	versionChangelog.Flags().StringVar(&config.branch, "branch", "master", "The branch to look for the start on")
	versionChangelog.Flags().StringVar(&config.fromTag, "from-tag", "", "If set only show commits after this tag (must be on the same branch)")
	versionChangelog.Flags().StringVar(&config.format, "format", string(FormatMarkdown), fmt.Sprintf("The output format (%s, %s)", FormatJson, FormatMarkdown))
	versionChangelog.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
	autoChangelog.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
}
