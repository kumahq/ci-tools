package main

import (
	"errors"
	"fmt"
	"github.com/kumahq/ci-tools/cmd/release-tool/internal/github"
	"github.com/spf13/cobra"
	"strings"
)

var githubReleaseChangelog = &cobra.Command{
	Use:   "changelog",
	Short: "create or update a release in github with the generated changelog",
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := fmt.Sprintf("release-%d.%d", major, minor)
		var prevVersion string
		if minor == 0 && patch == 0 { // We're shipping a new minor, we need to find the
			return fmt.Errorf("script doesn't work with new major versions")
		} else if patch == 0 {
			prevVersion = fmt.Sprintf("%d.%d.0", major, minor-1)
		} else {
			prevVersion = fmt.Sprintf("%d.%d.%d", major, minor, patch-1)
		}

		gqlClient := github.GqlClientFromEnv()
		releases, err := gqlClient.ReleaseGraphQL(config.repo)
		if err != nil {
			return err
		}
		header := `We are excited to announce the latest release !
TODO short description of the biggest features

## Notable Changes

TODO summary of some simple stuff.

## Changelog

`
		if patch != 0 {
			header = `This is a patch release that every user should upgrade to.

## Changelog

`
		}
		existingReleaseId := 0
		for _, r := range releases {
			if r.Name == config.release {
				existingReleaseId = r.Id
				if !r.IsDraft {
					return fmt.Errorf("release :%s has already published release notes, updating release-notes of released versions is not supported", config.release)
				}
				header = strings.SplitN(r.Description, "## Changelog", 2)[0] + "## Changelog\n\n"
				break
			}
		}
		sbuilder := &strings.Builder{}
		sbuilder.WriteString(header)
		changelog, err := getChangelog(gqlClient, config.repo, branch, prevVersion)
		if err != nil {
			return err
		}
		for _, v := range changelog {
			_, err = fmt.Fprintf(sbuilder, "* %s\n", v)
			if err != nil {
				return err
			}
		}

		return gqlClient.UpsertRelease(cmd.Context(), config.repo, config.release, sbuilder.String(), existingReleaseId)
	},
}

var major, minor, patch int
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
		major, minor, patch, err = config.SplitSemVer()
		return err
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("must pass a subcommand")
	},
}

func init() {
	githubReleaseChangelog.Flags().StringVar(&config.release, "release", "", "The name of the release to publish")
	githubReleaseChangelog.Flags().StringVar(&config.repo, "repo", "kumahq/kuma", "The repository to query")
	releaseCmd.AddCommand(githubReleaseChangelog)
}
