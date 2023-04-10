package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionChangelog)
	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(autoChangelog)
	rootCmd.AddCommand(versionFile)
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
