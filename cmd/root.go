package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/andrespd99/fireauth/internal/logger"
	"github.com/andrespd99/fireauth/internal/store"
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
	Use:     "fireauth",
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
}

// resolveVersion returns the effective version string. When the version was
// injected via ldflags at build time (anything other than "dev"), that wins.
// Otherwise it falls back to the VCS revision derived from Go's build info,
// which is available for binaries built with `go build`/`go install` (Go 1.18+
// embeds VCS metadata when the build is done inside a git repo).
func resolveVersion() string {
	if version != "dev" && version != "" {
		return version
	}
	if v, ok := vcsVersion(); ok {
		return v
	}
	return version
}

// vcsVersion reads the embedded VCS info from the build and returns a
// human-readable version string (tag if the commit is tagged, otherwise the
// short revision).
func vcsVersion() (string, bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}

	var revision, dirty string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if revision == "" {
		return "", false
	}
	short := revision
	if len(short) > 7 {
		short = short[:7]
	}
	return fmt.Sprintf("dev-%s%s", short, dirty), true
}

// Execute runs the root command.
func Execute() error {
	v := resolveVersion()
	rootCmd.Version = v
	rootCmd.SetVersionTemplate(fmt.Sprintf("fireauth %s\n", v))
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