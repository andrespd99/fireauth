package cmd

import (
	"fmt"
	"time"

	"github.com/andrespd99/fireauth/internal/store"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Show current project, session, and token status",
	Long:    "Display the active project, active session, and token expiry in a single line.",
	Example: `  fireauth status`,
	RunE:    runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	projectName, err := store.GetActiveProjectName()
	if err != nil {
		return err
	}

	p, err := store.LoadProject(projectName)
	if err != nil {
		return err
	}

	if p.ActiveSession == "" {
		fmt.Printf("Project: %s | No active session (run 'fireauth login')\n", projectName)
		return nil
	}

	sessions, err := store.LoadSessions(projectName)
	if err != nil {
		return err
	}

	sess, ok := sessions[p.ActiveSession]
	if !ok {
		fmt.Printf("Project: %s | Session: %s (not found — run 'fireauth login')\n", projectName, p.ActiveSession)
		return nil
	}

	remaining := time.Until(sess.TokenExpiry)
	tokenStatus := "valid"
	if remaining <= 0 {
		tokenStatus = "EXPIRED"
	} else if remaining < 5*time.Minute {
		tokenStatus = fmt.Sprintf("expiring (%s)", formatDuration(remaining))
	} else {
		tokenStatus = fmt.Sprintf("valid (%s remaining)", formatDuration(remaining))
	}

	fmt.Printf("Project: %s | Session: %s | Token: %s\n", projectName, sess.Email, tokenStatus)
	return nil
}
