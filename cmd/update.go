package cmd

import (
	"fmt"

	"github.com/andrespd99/fireauth/internal/logger"
	"github.com/andrespd99/fireauth/internal/updater"
	"github.com/spf13/cobra"
)

var flagCheck bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update fireauth to the latest version",
	Long:  "Check for a new release on GitHub and replace the current binary.",
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&flagCheck, "check", false, "only check for updates, don't install")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// 1. Resolve GitHub token.
	token, err := updater.ResolveToken()
	if err != nil {
		return err
	}

	// 2. Fetch latest release.
	release, err := updater.FetchLatestRelease(cmd.Context(), token)
	if err != nil {
		return err
	}

	// 3. Compare versions.
	if !updater.NeedsUpdate(version, release.TagName) {
		fmt.Printf("Already up to date (%s)\n", release.TagName)
		return nil
	}

	if flagCheck {
		fmt.Printf("Update available: %s → %s (run 'fireauth update' to install)\n", version, release.TagName)
		return nil
	}

	fmt.Printf("Updating fireauth %s → %s...\n", version, release.TagName)

	// 4. Find the right asset for this platform.
	goos, goarch := updater.CurrentPlatform()
	logger.Debug("platform detected", "os", goos, "arch", goarch)

	asset, err := updater.FindAsset(release, goos, goarch)
	if err != nil {
		return err
	}

	// 5. Download.
	archive, err := updater.DownloadAsset(cmd.Context(), token, asset.ID)
	if err != nil {
		return err
	}

	// 6. Extract binary from tar.gz.
	binary, err := updater.ExtractBinary(archive)
	if err != nil {
		return err
	}

	// 7. Replace current binary.
	if err := updater.Apply(binary); err != nil {
		return err
	}

	fmt.Printf("✓ Updated to %s\n", release.TagName)
	return nil
}
