package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/andrespd99/fireauth/internal/config"
	"github.com/andrespd99/fireauth/internal/logger"
	"github.com/andrespd99/fireauth/internal/store"
	"github.com/spf13/cobra"
)

var (
	flagAPIKey         string
	flagServiceAccount string
	flagProjectName    string
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Set up a Firebase project",
	Long:  "Interactive wizard that configures a Firebase project (API key and service account) for local use.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func init() {
	initCmd.Flags().StringVar(&flagAPIKey, "api-key", "", "Firebase Web API key (non-interactive)")
	initCmd.Flags().StringVar(&flagServiceAccount, "service-account", "", "path to Firebase service account JSON (non-interactive)")
	initCmd.Flags().StringVar(&flagProjectName, "name", "", "project name (defaults to the Firebase project_id from the service account)")
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

	// 2. Determine project name.
	projectName := flagProjectName
	if projectName == "" && len(args) == 1 {
		projectName = args[0]
	}

	// 3. Get API key.
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

	// 4. Get service account path.
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
	projectID, _ := saJSON["project_id"].(string)
	if projectID == "" {
		return fmt.Errorf("service account JSON missing 'project_id' field — is this the right file?")
	}
	logger.Debug("service account validated", "project_id", projectID)

	// Derive project name interactively if not provided.
	if projectName == "" {
		projects, _ := store.ListProjects()
		defaultName := "default"
		for _, p := range projects {
			if p == defaultName {
				defaultName = projectID
				break
			}
		}
		fmt.Printf("Project name [%s]: ", defaultName)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading project name: %w", err)
		}
		projectName = strings.TrimSpace(input)
		if projectName == "" {
			projectName = defaultName
		}
	}

	// 5. Create project directory and copy service account into it.
	pdir, err := config.ProjectDir(projectName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(pdir, 0700); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}

	destPath := filepath.Join(pdir, "service-account.json")
	if err := os.WriteFile(destPath, saData, 0600); err != nil {
		return fmt.Errorf("copying service account: %w", err)
	}
	logger.Debug("service account copied", "dest", destPath)

	// 6. Save project config.
	p := &store.Project{
		Name:               projectName,
		FirebaseAPIKey:     apiKey,
		ServiceAccountPath: destPath,
	}
	if err := store.SaveProject(p); err != nil {
		return fmt.Errorf("saving project config: %w", err)
	}

	// 7. Update global config (set active project, migrating legacy if needed).
	cfg, err := store.LoadConfig()
	if err != nil {
		// No config exists yet — create one.
		cfg = &store.Config{}
	}
	cfg.ActiveProject = projectName
	// Clear legacy fields if present.
	cfg.FirebaseAPIKey = ""
	cfg.ServiceAccountPath = ""
	cfg.ActiveSession = ""
	if err := store.SaveConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Check if this is the only/first project.
	projects, _ := store.ListProjects()

	fmt.Println()
	fmt.Println("✓ fireauth initialized successfully!")
	fmt.Printf("  Project:          %s\n", projectName)
	fmt.Printf("  Firebase Project: %s\n", projectID)
	fmt.Printf("  Config directory: %s\n", dir)
	fmt.Printf("  Active project:   %s\n", projectName)
	if len(projects) > 1 {
		fmt.Println()
		fmt.Println("Other projects:")
		for _, name := range projects {
			if name != projectName {
				fmt.Printf("  - %s\n", name)
			}
		}
	}
	fmt.Println()
	fmt.Println("Next step: run 'fireauth login' to sign in.")
	return nil
}