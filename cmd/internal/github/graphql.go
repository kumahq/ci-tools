package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v50/github"
)

type GQLOutput struct {
	Data GQLData `json:"data"`
}
type GQLData struct {
	Repository GQLRepo `json:"repository"`
}

type GQLPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type GQLRepo struct {
	Ref      GQLRef           `json:"ref"`
	Object   GQLObjectRepo    `json:"object"`
	Releases GQLObjectRelease `json:"releases"`
}

type GQLObjectRelease struct {
	Nodes []GQLRelease `json:"nodes"`
}

type GQLRelease struct {
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"createdAt"`
	PublishedAt  time.Time `json:"publishedAt"`
	IsDraft      bool      `json:"isDraft"`
	IsPrerelease bool      `json:"isPrerelease"`
	Description  string    `json:"description"`
	Id           int       `json:"databaseId"`
	IsLatest     bool      `json:"isLatest"`
}

func (r GQLRelease) SemVer() *semver.Version {
	return semver.MustParse(strings.TrimPrefix(r.Name, "v"))
}

// IsReleased returns if the release in not prerelease not a draft
func (r GQLRelease) IsReleased() bool {
	return !r.IsDraft && !r.IsPrerelease
}

var releasedOnRegexp = regexp.MustCompile("^> Released on ([0-9]{4}/[0-9]{2}/[0-9]{2})")

// ExtractReleaseDate returns the date this was published, if there's a `> Released on YYYY/MM/DD` in the description it uses this,
// otherwise it uses the PublishedAt data from Github.
func (r GQLRelease) ExtractReleaseDate() (time.Time, error) {
	res := releasedOnRegexp.FindStringSubmatch(r.Description)
	if len(res) == 2 {
		return time.Parse("2006/01/02", res[1])
	}
	return r.PublishedAt, nil
}

// Branch branch that this release was first on
func (r GQLRelease) Branch() string {
	// In theory we could extract this from the tag but let's keep this simple
	v := r.SemVer()
	return fmt.Sprintf("release-%d.%d", v.Major(), v.Minor())
}

type GQLObjectRepo struct {
	History GQLHistoryRepo `json:"history"`
}

type GQLHistoryRepo struct {
	PageInfo GQLPageInfo `json:"pageInfo"`
	Nodes    []GQLCommit `json:"nodes"`
}

type GQLAssociatedPRs struct {
	Nodes []GQLPRNode `json:"nodes"`
}

type GQLAuthor struct {
	Login string `json:"login"`
}

type GQLPRNode struct {
	Author GQLAuthor `json:"author"`
	Number int       `json:"number"`
	Title  string    `json:"title"`
	Body   string    `json:"body"`
}

type GQLRef struct {
	Target GQLRefTarget `json:"target"`
}

type GQLCommit struct {
	Oid                    string           `json:"oid"`
	Message                string           `json:"message"`
	AssociatedPullRequests GQLAssociatedPRs `json:"associatedPullRequests"`
}

type GQLRefTarget struct {
	CommitUrl string `json:"commitUrl"`
	Oid       string `json:"oid"`
}

func (r GQLRefTarget) Commit() string {
	return r.CommitUrl[strings.LastIndex(r.CommitUrl, "/")+1:]
}

type GQLClient struct {
	Token string
	Cl    *github.Client
}

func SplitRepo(repo string) (string, string) {
	r := strings.Split(repo, "/")
	return r[0], r[1]
}

func GqlClientFromEnv() *GQLClient {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_API_TOKEN")
		if token == "" {
			panic("need to set at least env GITHUB_TOKEN or GITHUB_API_TOKEN")
		}
	}
	cl := github.NewTokenClient(context.Background(), token)
	return &GQLClient{Token: token, Cl: cl}
}

func (c GQLClient) ReleaseGraphQL(repo string) ([]GQLRelease, error) {
	owner, name := SplitRepo(repo)
	res, err := c.graphqlQuery(`
query($name: String!, $owner: String!) {
  repository(owner: $owner, name: $name) {
    releases(first: 100, orderBy: {field: CREATED_AT, direction: DESC}) {
      nodes {
        name
        createdAt
        publishedAt
        isDraft
        isPrerelease
        description
        databaseId
        isLatest
      }
    }
  }
}
`, map[string]interface{}{"owner": owner, "name": name})
	if err != nil {
		return nil, err
	}
	return res.Data.Repository.Releases.Nodes, nil
}

func (c GQLClient) HistoryGraphQl(repo, branch, commitLimit string) ([]GQLCommit, error) {
	owner, name := SplitRepo(repo)
	var out []GQLCommit
	var err error
	var res GQLOutput
	for {
		cursorStr := ""
		if res.Data.Repository.Object.History.PageInfo.EndCursor != "" {
			cursorStr = fmt.Sprintf(`(after: "%s")`, res.Data.Repository.Object.History.PageInfo.EndCursor)
		}
		res, err = c.graphqlQuery(fmt.Sprintf(`
query($name: String!, $owner: String!, $branch: String!) {
  repository(owner: $owner, name: $name) {
    object(expression: $branch) {
      ... on Commit {
        history%s {
          pageInfo {
            hasNextPage
            endCursor
          }
          nodes {
            oid
            message
            associatedPullRequests(first: 1) {
              nodes {
                author {
                  login
                }
                number
                title
                body
              }
            }
          }
        }
      }
    }
  }
}
`, cursorStr), map[string]interface{}{"owner": owner, "name": name, "branch": branch})
		if err != nil {
			return out, err
		}
		for _, r := range res.Data.Repository.Object.History.Nodes {
			if commitLimit != "" && strings.HasPrefix(r.Oid, commitLimit) {
				return out, err
			}
			out = append(out, r)
		}
		if !res.Data.Repository.Object.History.PageInfo.HasNextPage {
			return out, err
		}
	}
}

func (c GQLClient) CommitByRef(repo, tag string) (string, error) {
	owner, name := SplitRepo(repo)
	res, err := c.graphqlQuery(`
query ($owner: String!, $name: String!, $ref: String!) {
  repository(name: $name, owner: $owner) {
    ref(qualifiedName: $ref) {
      target {
        commitUrl
        oid
      }
    }
  }
}
`, map[string]interface{}{"owner": owner, "name": name, "ref": fmt.Sprintf("refs/tags/%s", tag)})
	if err != nil {
		return "", err
	}
	// In some cases the oid doesn't match the github commit so let's extract the commit from the url.
	return res.Data.Repository.Ref.Target.Commit(), nil
}

func (c GQLClient) graphqlQuery(query string, variables map[string]interface{}) (GQLOutput, error) {
	var out GQLOutput
	var err error
	b2 := bytes.Buffer{}
	err = json.NewEncoder(&b2).Encode(map[string]interface{}{"query": query, "variables": variables})
	if err != nil {
		return out, err
	}
	var r *http.Request
	r, err = http.NewRequest(http.MethodPost, "https://api.github.com/graphql", &b2)
	if err != nil {
		return out, err
	}
	r.Header.Set("Authorization", fmt.Sprintf("bearer %s", c.Token))
	r.Header.Set("Content-Type", "application/json")
	var res *http.Response
	res, err = http.DefaultClient.Do(r)
	if err != nil {
		return out, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		b, _ := io.ReadAll(res.Body)
		err = fmt.Errorf("got status: %d body:%s", res.StatusCode, b)
		return out, err
	}
	err = json.NewDecoder(res.Body).Decode(&out)
	return out, err
}

func (c GQLClient) UpsertRelease(ctx context.Context, repo string, release string, contentModifier func(repositoryRelease *github.RepositoryRelease) error) error {
	releases, err := c.ReleaseGraphQL(repo)
	if err != nil {
		return err
	}
	var existingRelease *GQLRelease
	for _, r := range releases {
		if r.Name == release {
			existingRelease = &r
			break
		}
	}
	owner, name := SplitRepo(repo)
	if existingRelease == nil {
		releasePayload := &github.RepositoryRelease{Name: &release, Draft: github.Bool(true), TagName: &release}
		err := contentModifier(releasePayload)
		if err != nil {
			return err
		}
		_, _, err = c.Cl.Repositories.CreateRelease(ctx, owner, name, releasePayload)
		return err
	}
	releasePayload, _, err := c.Cl.Repositories.GetRelease(ctx, owner, name, int64(existingRelease.Id))
	if err != nil {
		return err
	}
	err = contentModifier(releasePayload)
	if err != nil {
		return err
	}
	_, _, err = c.Cl.Repositories.EditRelease(ctx, owner, name, int64(existingRelease.Id), releasePayload)
	return err
}
