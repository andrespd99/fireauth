package cmd

import (
	"fmt"

	"github.com/cashea-bnpl/auth-devtools/internal/logger"
	"github.com/spf13/cobra"
)

var (
	verbose bool
	version = "dev"
)

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
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")
	rootCmd.SetVersionTemplate(fmt.Sprintf("cashea-auth %s\n", version))
}

// Execute runs the root command.
func Execute() error {
	rootCmd.Version = version
	return rootCmd.Execute()
}
