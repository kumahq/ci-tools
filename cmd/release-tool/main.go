package main

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
)

func init() {
	rootCmd.AddCommand(versionChangelog)
	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(autoChangelog)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var config Config

type Config struct {
	branch  string
	repo    string
	fromTag string
	format  string
	release string
}

func SplitSemVer(in string) (int, int, int, error) {
	sp := strings.Split(in, ".")
	if len(sp) != 3 {
		return 0, 0, 0, fmt.Errorf("%s is not a valid semver", in)
	}
	major, err := strconv.Atoi(sp[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("major part of semver is not a number")
	}
	minor, err := strconv.Atoi(sp[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("minor part of semver is not a number")
	}
	patch, err := strconv.Atoi(sp[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("patch part of semver is not a number")
	}
	return major, minor, patch, nil
}

func MustSplitSemVer(in string) (int, int, int) {
	major, minor, patch, err := SplitSemVer(in)
	if err != nil {
		panic(err)
	}
	return major, minor, patch
}

var rootCmd = &cobra.Command{
	Use:   "release-tool",
	Short: "Do a lot of possible release fun",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if config.repo == "" {
			return errors.New("Must set a repo!")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("must pass a subcommand")
	},
}
