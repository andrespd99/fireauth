package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/andrespd99/fireauth/internal/logger"
	"github.com/andrespd99/fireauth/internal/store"
	"github.com/andrespd99/fireauth/internal/tui"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage Firebase projects",
	Long:  "Add, list, switch, and remove Firebase projects for different environments.",
}

var projectListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all configured projects",
	RunE:    runProjectList,
}

var projectUseCmd = &cobra.Command{
	Use:   "use [name]",
	Short: "Switch the active project",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runProjectUse,
}

var projectRemoveCmd = &cobra.Command{
	Use:     "remove [name]",
	Aliases: []string{"rm"},
	Short:   "Remove a configured project",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runProjectRemove,
}

var projectRenameCmd = &cobra.Command{
	Use:     "rename <old> <new>",
	Aliases: []string{"mv"},
	Short:   "Rename a project",
	Args:    cobra.ExactArgs(2),
	RunE:    runProjectRename,
}

var projectUpdateKeyCmd = &cobra.Command{
	Use:   "update-key [name]",
	Short: "Update the Firebase Web API key for a project",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runProjectUpdateKey,
}

var flagNewAPIKey string

func init() {
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectUseCmd)
	projectCmd.AddCommand(projectRemoveCmd)
	projectCmd.AddCommand(projectRenameCmd)
	projectCmd.AddCommand(projectUpdateKeyCmd)
	projectUpdateKeyCmd.Flags().StringVar(&flagNewAPIKey, "api-key", "", "new Firebase Web API key (non-interactive)")
	rootCmd.AddCommand(projectCmd)
}

// --- project list ---

func runProjectList(cmd *cobra.Command, args []string) error {
	projects, err := store.ListProjects()
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		fmt.Println("No projects configured. Run 'fireauth init' to add one.")
		return nil
	}

	activeProject, err := store.GetActiveProjectName()
	if err != nil {
		logger.Warn("failed to get active project", "error", err)
	}

	sort.Strings(projects)

	fmt.Println()
	fmt.Printf("  %-20s %s\n", "PROJECT", "")
	fmt.Printf("  %-20s %s\n", "-------", "")
	for _, name := range projects {
		marker := "  "
		if name == activeProject {
			marker = "← active"
		}
		p, err := store.LoadProject(name)
		if err != nil {
			fmt.Printf("  %-20s (error: %v)\n", name, err)
			continue
		}
		info := ""
		if p.ActiveSession != "" {
			info = fmt.Sprintf("session: %s  ", p.ActiveSession)
		}
		fmt.Printf("  %-20s %s%s\n", name, info, marker)
	}
	fmt.Println()
	return nil
}

// --- project use ---

func runProjectUse(cmd *cobra.Command, args []string) error {
	projects, err := store.ListProjects()
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		return fmt.Errorf("no projects configured — run 'fireauth init' first")
	}

	var target string
	if len(args) == 1 {
		target = args[0]
	} else {
		sort.Strings(projects)

		activeProject, err := store.GetActiveProjectName()
		if err != nil {
			logger.Warn("failed to get active project", "error", err)
		}

		selected, err := tui.Pick("Select project", projects, activeProject)
		if err != nil {
			return err
		}
		if selected == "" {
			fmt.Fprintln(os.Stderr, "cancelled")
			return nil
		}
		target = selected
	}

	// Validate the project exists.
	if !contains(projects, target) {
		return fmt.Errorf("project %q not found — available: %s", target, strings.Join(projects, ", "))
	}

	if err := store.SetActiveProject(target); err != nil {
		return err
	}

	logger.Debug("switched active project", "project", target)
	fmt.Printf("✓ Switched to project %s\n", target)

	// Show active session if any.
	p, err := store.LoadProject(target)
	if err == nil && p.ActiveSession != "" {
		fmt.Printf("  Active session: %s\n", p.ActiveSession)
	}
	return nil
}

// --- project remove ---

func runProjectRemove(cmd *cobra.Command, args []string) error {
	projects, err := store.ListProjects()
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		return fmt.Errorf("no projects configured")
	}

	var target string
	if len(args) == 1 {
		target = args[0]
	} else {
		sort.Strings(projects)

		activeProject, err := store.GetActiveProjectName()
		if err != nil {
			logger.Warn("failed to get active project", "error", err)
		}

		selected, err := tui.Pick("Select project to remove", projects, activeProject)
		if err != nil {
			return err
		}
		if selected == "" {
			fmt.Fprintln(os.Stderr, "cancelled")
			return nil
		}
		target = selected
	}

	// Validate.
	if !contains(projects, target) {
		return fmt.Errorf("project %q not found", target)
	}

	wasActive := false
	activeProject, err := store.GetActiveProjectName()
	if err != nil {
		logger.Warn("failed to get active project", "error", err)
	}
	if activeProject == target {
		wasActive = true
	}

	if err := store.DeleteProject(target); err != nil {
		return fmt.Errorf("removing project: %w", err)
	}

	remaining, _ := store.ListProjects()
	if len(remaining) == 0 {
		if wasActive {
			if err := store.ClearActiveProject(); err != nil {
				return err
			}
		}
		fmt.Printf("✓ Removed project %s\n", target)
		fmt.Println("No projects remaining. Run 'fireauth init' to add one.")
		return nil
	}

	if wasActive {
		sort.Strings(remaining)
		if err := store.SetActiveProject(remaining[0]); err != nil {
			return err
		}
		fmt.Printf("  Active project switched to %s\n", remaining[0])
	}

	fmt.Printf("✓ Removed project %s\n", target)
	return nil
}

// --- project rename ---

func runProjectRename(cmd *cobra.Command, args []string) error {
	oldName := args[0]
	newName := args[1]

	// Validate the old project exists.
	projects, err := store.ListProjects()
	if err != nil {
		return err
	}
	if !contains(projects, oldName) {
		return fmt.Errorf("project %q not found — available: %s", oldName, strings.Join(projects, ", "))
	}

	if err := store.RenameProject(oldName, newName); err != nil {
		return err
	}

	fmt.Printf("✓ Renamed project %s → %s\n", oldName, newName)
	return nil
}

// --- project update-key ---

func runProjectUpdateKey(cmd *cobra.Command, args []string) error {
	projects, err := store.ListProjects()
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		return fmt.Errorf("no projects configured — run 'fireauth init' first")
	}

	var name string
	if len(args) == 1 {
		name = args[0]
	} else {
		active, err := resolveProjectName()
		if err == nil && active != "" {
			name = active
		} else {
			sort.Strings(projects)
			selected, err := tui.Pick("Select project", projects, active)
			if err != nil {
				return err
			}
			if selected == "" {
				fmt.Fprintln(os.Stderr, "cancelled")
				return nil
			}
			name = selected
		}
	}

	p, err := store.LoadProject(name)
	if err != nil {
		return err
	}

	apiKey := flagNewAPIKey
	if apiKey == "" {
		fmt.Printf("New Firebase Web API Key for %s: ", name)
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading API key: %w", err)
		}
		apiKey = strings.TrimSpace(input)
	}
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	p.FirebaseAPIKey = apiKey
	if err := store.SaveProject(p); err != nil {
		return fmt.Errorf("saving project: %w", err)
	}

	logger.Debug("updated API key", "project", name, "length", len(apiKey))
	fmt.Printf("✓ Updated API key for project %s\n", name)
	return nil
}

// --- helpers ---

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
