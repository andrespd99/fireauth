package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cashea-bnpl/auth-devtools/internal/config"
	"github.com/cashea-bnpl/auth-devtools/internal/logger"
	"github.com/cashea-bnpl/auth-devtools/internal/store"
	"github.com/spf13/cobra"
)

var (
	flagAPIKey         string
	flagServiceAccount string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up cashea-auth for the first time",
	Long:  "Interactive wizard that configures the Firebase API key and service account for local use.",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().StringVar(&flagAPIKey, "api-key", "", "Firebase Web API key (non-interactive)")
	initCmd.Flags().StringVar(&flagServiceAccount, "service-account", "", "path to Firebase service account JSON (non-interactive)")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	// 1. Ensure config directory.
	dir, err := config.EnsureDir()
	if err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	logger.Debug("config directory ready", "path", dir)

	// 2. Get API key.
	apiKey := flagAPIKey
	if apiKey == "" {
		fmt.Print("Firebase Web API Key: ")
		apiKey, err = reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading API key: %w", err)
		}
		apiKey = strings.TrimSpace(apiKey)
	}
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}
	logger.Debug("API key provided", "length", len(apiKey))

	// 3. Get service account path.
	saPath := flagServiceAccount
	if saPath == "" {
		fmt.Print("Path to Firebase service account JSON: ")
		saPath, err = reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading service account path: %w", err)
		}
		saPath = strings.TrimSpace(saPath)
	}

	// Expand ~ if present.
	if strings.HasPrefix(saPath, "~/") {
		home, _ := os.UserHomeDir()
		saPath = filepath.Join(home, saPath[2:])
	}

	// Validate the service account file.
	logger.Debug("validating service account", "path", saPath)
	saData, err := os.ReadFile(saPath)
	if err != nil {
		return fmt.Errorf("reading service account file: %w", err)
	}

	var saJSON map[string]interface{}
	if err := json.Unmarshal(saData, &saJSON); err != nil {
		return fmt.Errorf("service account file is not valid JSON: %w", err)
	}
	if _, ok := saJSON["project_id"]; !ok {
		return fmt.Errorf("service account JSON missing 'project_id' field — is this the right file?")
	}
	logger.Debug("service account validated", "project_id", saJSON["project_id"])

	// 4. Copy service account into config dir.
	destPath := filepath.Join(dir, "service-account.json")
	if err := os.WriteFile(destPath, saData, 0600); err != nil {
		return fmt.Errorf("copying service account: %w", err)
	}
	logger.Debug("service account copied", "dest", destPath)

	// 5. Save config.
	cfg := &store.Config{
		FirebaseAPIKey:     apiKey,
		ServiceAccountPath: destPath,
	}
	if err := store.SaveConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ cashea-auth initialized successfully!")
	fmt.Printf("  Config directory: %s\n", dir)
	fmt.Println()
	fmt.Println("Next step: run 'cashea-auth login' to sign in.")
	return nil
}
