package cmd

import (
	"fmt"

	"github.com/andrespd99/fireauth/internal/logger"
	"github.com/andrespd99/fireauth/internal/store"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout [email]",
	Short: "Remove a stored session",
	Long:  "Remove a stored session from the active project. If no email is given, removes the active session.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command, args []string) error {
	projectName, err := resolveProjectName()
	if err != nil {
		return err
	}

	p, err := store.LoadProject(projectName)
	if err != nil {
		return err
	}

	sessions, err := store.LoadSessions(projectName)
	if err != nil {
		return err
	}

	// Determine which email to remove.
	targetEmail := p.ActiveSession
	if len(args) == 1 {
		targetEmail = args[0]
	}

	if targetEmail == "" {
		return fmt.Errorf("no active session and no email provided")
	}

	if _, ok := sessions[targetEmail]; !ok {
		return fmt.Errorf("session %q not found in project %q", targetEmail, projectName)
	}

	// Remove the session.
	if err := store.DeleteSession(projectName, targetEmail); err != nil {
		return fmt.Errorf("removing session: %w", err)
	}
	logger.Debug("session removed", "project", projectName, "email", targetEmail)

	// Reload project to get updated active session.
	p, err = store.LoadProject(projectName)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Logged out %s\n", targetEmail)
	if p.ActiveSession != "" && p.ActiveSession != targetEmail {
		fmt.Printf("  Active session: %s\n", p.ActiveSession)
	} else if p.ActiveSession == "" {
		fmt.Println("  No sessions remaining.")
	}
	fmt.Printf("  Project: %s\n", projectName)

	return nil
}