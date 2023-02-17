package changeloggenerator

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

func New(repo string, input []CommitInfo) (Changelog, error) {
	byChangelog := map[string][]*CommitInfo{}
	// Rollup changes together
	for i := range input {
		if (&input[i]).normalize() {
			byChangelog[input[i].changelog] = append(byChangelog[input[i].changelog], &input[i])
		}
	}
	// Create a list to display
	var out []ChangelogItem
	for changelog, commits := range byChangelog {
		uniqueAuthors := map[string]interface{}{}
		uniquePrs := map[int]interface{}{}
		var authors []string
		var prs []int
		sort.Slice(commits, func(i, j int) bool {
			return commits[i].PrNumber < commits[j].PrNumber
		})
		var minVersion, maxVersion string
		for _, c := range commits {
			// Required because in the past we weren't squashing commits
			if _, exists := uniquePrs[c.PrNumber]; exists {
				continue
			}
			uniquePrs[c.PrNumber] = nil
			prs = append(prs, c.PrNumber)
			if minVersion == "" {
				minVersion = c.startDependency
			}
			maxVersion = c.endDependency
			if _, exists := uniqueAuthors[c.Author]; !exists {
				authors = append(authors, fmt.Sprintf("@%s", c.Author))
				uniqueAuthors[c.Author] = nil
			}
		}
		if minVersion != "" && maxVersion != "" {
			changelog = fmt.Sprintf("%s from %s to %s", changelog, minVersion, maxVersion)
		}
		sort.Strings(authors)
		out = append(out, ChangelogItem{Repo: repo, Desc: changelog, Authors: authors, PullRequests: prs})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Desc < out[j].Desc
	})
	return out, nil
}

type Changelog []ChangelogItem

type ChangelogItem struct {
	Desc         string   `json:"desc"`
	Authors      []string `json:"authors"`
	PullRequests []int    `json:"pull_requests"`
	Repo         string
}

func (c ChangelogItem) String() string {
	var prLinks []string
	for _, n := range c.PullRequests {
		prLinks = append(prLinks, fmt.Sprintf("[#%d](https://github.com/%s/pull/%d)", n, c.Repo, n))
	}
	seen := map[string]struct{}{}
	var authors []string
	for _, a := range c.Authors {
		if _, ok := seen[a]; !ok {
			authors = append(authors, a)
			seen[a] = struct{}{}
		}
	}
	sort.Strings(authors)
	return fmt.Sprintf("%s %s %s", c.Desc, strings.Join(prLinks, " "), strings.Join(authors, ","))
}

type CommitInfo struct {
	Sha             string
	Author          string
	PrNumber        int
	PrTitle         string
	PrBody          string
	CommitMessage   string
	changelog       string
	startDependency string
	endDependency   string
}

// titles look like: chore(deps): bump github.com/lib/pq from 1.10.6 to 1.10.7
var dependabotPRTitleRegExp = regexp.MustCompile(`(chore\(deps\): [bB]ump [^ ]+) from ([^ ]+) to ([^ ]+).*`)

func (ci *CommitInfo) normalize() bool {
	changelog := ""
	for _, l := range strings.Split(ci.PrBody, "\n") {
		if strings.HasPrefix(l, "> Changelog: ") {
			changelog = strings.TrimSpace(strings.TrimPrefix(l, "> Changelog: "))
		}
	}
	switch changelog {
	case "skip":
		return false
	case "":
		// Ignore prs with usually ignored prefix
		for _, v := range []string{"build", "ci", "test", "refactor", "fix(ci)", "fix(test)", "docs"} {
			if strings.HasPrefix(ci.CommitMessage, v) {
				return false
			}
		}
		// Only prs with chore(deps) are included
		if strings.HasPrefix(ci.CommitMessage, "chore") && !strings.HasPrefix(ci.CommitMessage, "chore(deps)") {
			return false
		}
		// Use the pr.Title as a changelog entry
		ci.changelog = ci.PrTitle
	default:
		ci.changelog = changelog
	}
	if matches := dependabotPRTitleRegExp.FindStringSubmatch(ci.changelog); matches != nil {
		// Rollup dependabot issues with the same dependency into just one so we can rebuild a single line with all update PRs.
		ci.changelog = strings.ReplaceAll(matches[1], "Bump", "bump")
		ci.startDependency = matches[2]
		ci.endDependency = matches[3]
	}
	return true
}
