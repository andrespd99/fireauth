package cmd

import (
	"fmt"

	"github.com/cashea-bnpl/auth-devtools/internal/logger"
	"github.com/cashea-bnpl/auth-devtools/internal/store"
	"github.com/spf13/cobra"
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
	Use:     "cashea-auth",
	Short:   "Firebase auth utilities for testing",
	Long:    "A CLI tool to authenticate against Firebase and manage bearer tokens for REST API testing.",
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.Init(verbose)
		// Migrate legacy single-project config → multi-project, if needed.
		// This is idempotent and only runs once.
		if cmd.Name() != "init" {
			_ = store.MigrateLegacyConfig()
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")
	rootCmd.PersistentFlags().StringVar(&flagProject, "project", "", "override the active project for this command")
	rootCmd.SetVersionTemplate(fmt.Sprintf("cashea-auth %s\n", version))
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