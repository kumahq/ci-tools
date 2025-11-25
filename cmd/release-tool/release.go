package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/Masterminds/semver/v3"
	github2 "github.com/google/go-github/v78/github"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"

	"github.com/kumahq/ci-tools/cmd/internal/github"
)

const (
	// GitHubMaxBodySize is the maximum allowed size for GitHub release body
	GitHubMaxBodySize = 125000
)

var (
	version *semver.Version
	dryRun  bool
)

var githubReleaseChangelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "create or update a release in github with the generated changelog",
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := fmt.Sprintf("release-%d.%d", version.Major(), version.Minor())
		var prevVersion *semver.Version
		if version.Major() == 0 && version.Minor() == 0 {
			return fmt.Errorf("script doesn't work with new major versions")
		} else if version.Patch() == 0 {
			prevVersion = semver.New(version.Major(), version.Minor()-1, 0, "", "")
		} else {
			prevVersion = semver.New(version.Major(), version.Minor(), version.Patch()-1, "", "")
		}

		header := `We are excited to announce the latest release !
TODO short description of the biggest features

## Notable Changes

TODO summary of some simple stuff.

## Changelog

`
		if version.Patch() != 0 {
			header = `This is a patch release that every user should upgrade to.

## Changelog

`
		}

		gqlClient, err := github.NewGQLClient(config.useGHAuth)
		if err != nil {
			return err
		}

		// Normalize the previous version tag for display and lookup
		prevTag := NormalizeVersionTag(prevVersion.String())

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "getting changelog from %s on repo %s and branch %s\n", prevTag, config.repo, branch)
		if err != nil {
			return err
		}

		// Use warnOnNormalize=false since prevVersion is auto-derived, not user-provided
		changelog, err := getChangelog(gqlClient, config.repo, branch, prevTag, false)
		if err != nil {
			return err
		}

		// Build the release body
		buildBody := func(existingBody *string) string {
			sbuilder := &strings.Builder{}
			if existingBody != nil {
				header = strings.SplitN(*existingBody, "## Changelog", 2)[0] + "## Changelog\n\n"
			}

			sbuilder.WriteString(header)

			for _, v := range changelog {
				_, _ = fmt.Fprintf(sbuilder, "* %s\n", v)
			}

			return sbuilder.String()
		}

		// For dry-run, build and display the body without touching GitHub
		if dryRun {
			body := buildBody(nil)
			bodyLen := len(body)

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n--- Release Body Preview (%d characters) ---\n", bodyLen)
			_, _ = fmt.Fprint(cmd.OutOrStdout(), body)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "--- End Preview ---\n\n")

			if bodyLen > GitHubMaxBodySize {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "⚠️  WARNING: Body exceeds GitHub limit of %d characters by %d characters\n", GitHubMaxBodySize, bodyLen-GitHubMaxBodySize)
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✅ Body size OK: %d/%d characters (%.1f%% of limit)\n", bodyLen, GitHubMaxBodySize, float64(bodyLen)/float64(GitHubMaxBodySize)*100)
			}

			return nil
		}

		if len(changelog) == 0 {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "no changelog\n")
			return err
		}

		// Normalize release tag to match kumahq/kuma Git tag format
		// Use WithWarning since config.release is user-provided
		releaseTag := NormalizeVersionTagWithWarning(config.release)
		// Release name should not have v prefix (just the version number)
		releaseName := strings.TrimPrefix(releaseTag, "v")

		return gqlClient.UpsertRelease(cmd.Context(), config.repo, releaseName, releaseTag, func(release *github2.RepositoryRelease) error {
			if !release.GetDraft() {
				return fmt.Errorf("release :%s has already published release notes, updating release-notes of released versions is not supported", release)
			}

			body := buildBody(release.Body)

			// Check body size and fail with helpful message if too large
			if len(body) > GitHubMaxBodySize {
				return fmt.Errorf("release body exceeds GitHub limit: %d characters (max %d). Use --dry-run to preview the body and consider manually truncating", len(body), GitHubMaxBodySize)
			}

			// Normalize release name to not have v prefix (SLSA provenance may create releases with v prefix)
			release.Name = github2.Ptr(releaseName)
			release.Body = github2.Ptr(body)

			return nil
		})
	},
}

var helmChartCmd = &cobra.Command{
	Use:   "helm-chart",
	Short: "add a reference to the helm chart in the release notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		if chartRepo == "" {
			return errors.New("must set --charts-repo")
		}

		gqlClient, err := github.NewGQLClient(config.useGHAuth)
		if err != nil {
			return err
		}

		releases, err := gqlClient.ReleaseGraphQL(chartRepo)
		if err != nil {
			return err
		}
		// Strip v-prefix from release version to match helm chart naming convention
		// Git tags use v-prefix (v2.11.8) but helm charts don't (kuma-2.11.8)
		releaseVersion := strings.TrimPrefix(config.release, "v")
		expectedName := fmt.Sprintf("%s-%s", strings.Split(config.repo, "/")[1], releaseVersion)
		var release *github.GQLRelease
		for _, r := range releases {
			if r.Name == expectedName {
				release = &r
				break
			}
		}
		if release == nil {
			return errors.New("couldn't find matching helm charts")
		}
		_, _ = cmd.OutOrStdout().Write([]byte("Found helm chart"))
		// TODO we could update the release with a link to artifactory
		return nil
	},
}

var binariesCmd = &cobra.Command{
	Use:   "binaries",
	Short: "Check all binaries are present in the right place",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(binaries) == 0 {
			return errors.New("need to specific at least one binary")
		}
		var merr *multierror.Error
		org, name := github.SplitRepo(config.repo)
		tmpl, err := template.New("").Parse(urlTemplate)
		if err != nil {
			return err
		}
		// Strip v-prefix from release version to match binary naming convention
		releaseVersion := strings.TrimPrefix(config.release, "v")
		for _, binary := range binaries {
			buf := bytes.NewBuffer(nil)
			err := tmpl.Execute(buf, struct {
				Org     string
				Repo    string
				Binary  string
				Release string
			}{
				Org: org, Repo: name, Binary: binary, Release: releaseVersion,
			})
			if err != nil {
				return err
			}
			u := buf.String()
			if err != nil {
				merr = multierror.Append(merr, fmt.Errorf("couldn't join url path: %w", err))
				continue
			}
			r, err := http.Get(u)
			if err != nil {
				merr = multierror.Append(merr, fmt.Errorf("couldn't get %s: %w", u, err))
			} else if r.StatusCode != 200 {
				merr = multierror.Append(merr, fmt.Errorf("couldn't get %s: %d", u, r.StatusCode))
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Found: %s\n", u)
			}
			_ = r.Body.Close()
		}
		return merr.ErrorOrNil()
	},
}

var (
	dockerImages     []string
	dockerRepository string
	dockerCmd        = &cobra.Command{
		Use:   "docker",
		Short: "Check all images",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(dockerImages) == 0 {
				return errors.New("need to specify some docker images")
			}
			if dockerRepository == "" {
				return errors.New("need to specify a docker repository")
			}
			// Strip v-prefix from release version to match Docker tag naming convention
			releaseVersion := strings.TrimPrefix(config.release, "v")
			var merr *multierror.Error
			for _, i := range dockerImages {
				img := fmt.Sprintf("%s/%s:%s", dockerRepository, i, releaseVersion)
				r, err := http.Head(fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/%s/tags/%s", dockerRepository, i, releaseVersion))
				if err != nil {
					merr = multierror.Append(merr, fmt.Errorf("failed with image: %s %w", img, err))
				} else if r.StatusCode != 200 {
					merr = multierror.Append(merr, fmt.Errorf("failed with image: %s status: %d", img, r.StatusCode))
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Got image: %s\n", img)
				}
			}
			return merr.ErrorOrNil()
		},
	}
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Do a lot of possible release fun",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if config.repo == "" {
			return errors.New("you must have a valid `--repo`")
		}
		if config.release == "" {
			return errors.New("you must set `--release`")
		}

		var err error
		version, err = semver.NewVersion(config.release)
		return err
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("must pass a subcommand")
	},
}

var (
	binaries    []string
	chartRepo   string
	urlTemplate string
)

func init() {
	githubReleaseChangelogCmd.Flags().StringVar(&config.release, "release", "", "The name of the release to publish")
	githubReleaseChangelogCmd.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
	githubReleaseChangelogCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview the release body without updating GitHub")
	helmChartCmd.Flags().StringVar(&chartRepo, "charts-repo", "", "The repository to query")
	helmChartCmd.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
	helmChartCmd.Flags().StringVar(&config.release, "release", "", "The name of the release to publish")

	binariesCmd.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
	binariesCmd.Flags().StringVar(&config.release, "release", "", "The name of the release to publish")
	binariesCmd.Flags().StringSliceVar(&binaries, "binaries", binaries, "A comma separated list of targets (.e.g: centos-amd64,darwin-arm64)")
	binariesCmd.Flags().StringVar(&urlTemplate, "url-template", "https://packages.konghq.com/public/{{.Repo}}-binaries-release/raw/names/{{.Repo}}-{{.Binary}}/versions/{{.Release}}/{{.Repo}}-{{.Release}}-{{.Binary}}.tar.gz", "A template to use for the binary")

	dockerCmd.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
	dockerCmd.Flags().StringVar(&config.release, "release", "", "The name of the release to publish")
	dockerCmd.Flags().StringVar(&dockerRepository, "docker-repo", "", "The name of the docker repo")
	dockerCmd.Flags().StringSliceVar(&dockerImages, "images", dockerImages, "A comma separated list of images (.e.g: kumactl,kuma-cp)")

	releaseCmd.AddCommand(githubReleaseChangelogCmd)
	releaseCmd.AddCommand(helmChartCmd)
	releaseCmd.AddCommand(binariesCmd)
	releaseCmd.AddCommand(dockerCmd)
}
