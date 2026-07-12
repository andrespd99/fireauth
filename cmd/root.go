package cmd

import (
	"fmt"
	"os"

	"github.com/andrespd99/fireauth/internal/logger"
	"github.com/andrespd99/fireauth/internal/store"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	verbose bool
	version = "dev"
)

// flagProject overrides the active project for a single command invocation.
var flagProject string

// SetVersion is called from main to inject the build-time version.
func SetVersion(v string) {
	version = v
}

var rootCmd = &cobra.Command{
	Use:          "fireauth",
	Short:        "Firebase auth utilities for testing",
	Long:         "A CLI tool to authenticate against Firebase and manage bearer tokens for REST API testing.",
	Version:      version,
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.Init(verbose)
		// Migrate legacy single-project config → multi-project, if needed.
		// This is idempotent and only runs once.
		if cmd.Name() != "init" {
			if err := store.MigrateLegacyConfig(); err != nil {
				logger.Warn("legacy config migration failed", "error", err)
			}
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")
	rootCmd.PersistentFlags().StringVarP(&flagProject, "project", "p", "", "override the active project for this command")
	rootCmd.SetVersionTemplate(fmt.Sprintf("fireauth %s\n", version))
}

// Execute runs the root command.
func Execute() error {
	rootCmd.Version = version
	return rootCmd.Execute()
}

// resolveProjectName returns the project to use for the current command. If
// the --project flag is set, it takes precedence; otherwise the active
// project from config is used.
func resolveProjectName() (string, error) {
	if flagProject != "" {
		return flagProject, nil
	}
	return store.GetActiveProjectName()
}

// isTerminal returns true if stdin is a terminal (used to decide whether to
// show interactive prompts).
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
