package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/andrespd99/fireauth/internal/logger"
	"github.com/andrespd99/fireauth/internal/store"
	"github.com/andrespd99/fireauth/internal/tui"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch [email]",
	Short: "Switch the active session",
	Long:  "Switch the active session to a different stored user within the active project. If no email is provided, an interactive picker is shown.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSwitch,
}

func init() {
	rootCmd.AddCommand(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) error {
	projectName, err := resolveProjectName()
	if err != nil {
		return err
	}

	sessions, err := store.LoadSessions(projectName)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		return fmt.Errorf("no sessions stored in project %q — run 'fireauth login' first", projectName)
	}

	var targetEmail string

	if len(args) == 1 {
		targetEmail = args[0]
	} else {
		emails := make([]string, 0, len(sessions))
		for email := range sessions {
			emails = append(emails, email)
		}
		sort.Strings(emails)

		p, err := store.LoadProject(projectName)
		if err != nil {
			return err
		}

		selected, err := tui.Pick("Select session", emails, p.ActiveSession)
		if err != nil {
			return err
		}
		if selected == "" {
			fmt.Fprintln(os.Stderr, "cancelled")
			return nil
		}
		targetEmail = selected
	}

	// Validate the target exists.
	if _, ok := sessions[targetEmail]; !ok {
		return fmt.Errorf("session %q not found — available: %s", targetEmail, availableEmails(sessions))
	}

	if err := store.SetActiveSession(projectName, targetEmail); err != nil {
		return err
	}

	logger.Debug("switched active session", "project", projectName, "email", targetEmail)
	fmt.Printf("✓ Switched to %s (project: %s)\n", targetEmail, projectName)
	return nil
}

func availableEmails(sessions store.Sessions) string {
	emails := make([]string, 0, len(sessions))
	for email := range sessions {
		emails = append(emails, email)
	}
	sort.Strings(emails)
	return strings.Join(emails, ", ")
}