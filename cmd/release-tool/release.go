package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Masterminds/semver/v3"
	github2 "github.com/google/go-github/v50/github"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"

	"github.com/kumahq/ci-tools/cmd/internal/github"
)

var version *semver.Version

var githubReleaseChangelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "create or update a release in github with the generated changelog",
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := fmt.Sprintf("release-%d.%d", version.Major(), version.Minor())
		var prevVersion *semver.Version
		if version.Major() == 0 && version.Minor() == 0 { // We're shipping a new minor, we need to find the
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

		gqlClient := github.GqlClientFromEnv()
		return gqlClient.UpsertRelease(cmd.Context(), config.repo, config.release, func(release *github2.RepositoryRelease) error {
			if !release.GetDraft() {
				return fmt.Errorf("release :%s has already published release notes, updating release-notes of released versions is not supported", release)
			}
			sbuilder := &strings.Builder{}
			if release.Body != nil {
				header = strings.SplitN(release.GetBody(), "## Changelog", 2)[0] + "## Changelog\n\n"
			}
			sbuilder.WriteString(header)
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "getting changelog from %s on repo %s and branch %s\n", prevVersion, config.repo, branch)
			if err != nil {
				return err
			}
			changelog, err := getChangelog(gqlClient, config.repo, branch, prevVersion.String())
			if err != nil {
				return err
			}
			if len(changelog) == 0 {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "no changelog\n")
				return err
			}
			for _, v := range changelog {
				_, err = fmt.Fprintf(sbuilder, "* %s\n", v)
				if err != nil {
					return err
				}
			}
			release.Body = github2.String(sbuilder.String())
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
		gqlClient := github.GqlClientFromEnv()
		releases, err := gqlClient.ReleaseGraphQL(chartRepo)
		if err != nil {
			return err
		}
		expectedName := fmt.Sprintf("%s-%s", strings.Split(config.repo, "/")[1], config.release)
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

var pulpCmd = &cobra.Command{
	Use:   "pulp-binaries",
	Short: "Check all binaries are present on pulp",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(binaries) == 0 {
			return errors.New("need to specific at least one binary")
		}
		var merr *multierror.Error
		_, name := github.SplitRepo(config.repo)
		for _, binary := range binaries {
			u, err := url.JoinPath(pulpUrl, fmt.Sprintf("/%s-%s-%s.tar.gz", name, config.release, binary))
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
				_, _ = cmd.OutOrStdout().Write([]byte(fmt.Sprintf("Found: %s\n", u)))
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
			var merr *multierror.Error
			for _, i := range dockerImages {
				img := fmt.Sprintf("%s/%s:%s", dockerRepository, i, config.release)
				r, err := http.Head(fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/%s/tags/%s", dockerRepository, i, config.release))
				if err != nil {
					merr = multierror.Append(merr, fmt.Errorf("failed with image: %s %w", img, err))
				} else if r.StatusCode != 200 {
					merr = multierror.Append(merr, fmt.Errorf("failed with image: %s status: %d", img, r.StatusCode))
				} else {
					_, _ = cmd.OutOrStdout().Write([]byte(fmt.Sprintf("Got image: %s\n", img)))
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
	binaries  []string
	pulpUrl   string
	chartRepo string
)

func init() {
	githubReleaseChangelogCmd.Flags().StringVar(&config.release, "release", "", "The name of the release to publish")
	githubReleaseChangelogCmd.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
	helmChartCmd.Flags().StringVar(&chartRepo, "charts-repo", "", "The repository to query")
	helmChartCmd.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
	helmChartCmd.Flags().StringVar(&config.release, "release", "", "The name of the release to publish")

	pulpCmd.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
	pulpCmd.Flags().StringVar(&config.release, "release", "", "The name of the release to publish")
	pulpCmd.Flags().StringSliceVar(&binaries, "binaries", binaries, "A comma separated list of targets (.e.g: centos-amd64,darwin-arm64)")
	pulpCmd.Flags().StringVar(&pulpUrl, "base-url", "https://download.konghq.com/mesh-alpine/", "The url where releases are published")

	dockerCmd.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
	dockerCmd.Flags().StringVar(&config.release, "release", "", "The name of the release to publish")
	dockerCmd.Flags().StringVar(&dockerRepository, "docker-repo", "", "The name of the docker repo")
	dockerCmd.Flags().StringSliceVar(&dockerImages, "images", dockerImages, "A comma separated list of images (.e.g: kumactl,kuma-cp)")

	releaseCmd.AddCommand(githubReleaseChangelogCmd)
	releaseCmd.AddCommand(helmChartCmd)
	releaseCmd.AddCommand(pulpCmd)
	releaseCmd.AddCommand(dockerCmd)
}
