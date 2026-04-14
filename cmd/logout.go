package cmd

import (
	"fmt"

	"github.com/cashea-bnpl/auth-devtools/internal/logger"
	"github.com/cashea-bnpl/auth-devtools/internal/store"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout [email]",
	Short: "Remove a stored session",
	Long:  "Remove a stored session. If no email is given, removes the active session.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command, args []string) error {
	cfg, err := store.LoadConfig()
	if err != nil {
		return err
	}

	sessions, err := store.LoadSessions()
	if err != nil {
		return err
	}

	// Determine which email to remove.
	targetEmail := cfg.ActiveSession
	if len(args) == 1 {
		targetEmail = args[0]
	}

	if targetEmail == "" {
		return fmt.Errorf("no active session and no email provided")
	}

	if _, ok := sessions[targetEmail]; !ok {
		return fmt.Errorf("session %q not found", targetEmail)
	}

	// Remove the session.
	delete(sessions, targetEmail)
	if err := store.SaveSessions(sessions); err != nil {
		return fmt.Errorf("saving sessions: %w", err)
	}
	logger.Debug("session removed", "email", targetEmail)

	// Update active session if we just removed it.
	if cfg.ActiveSession == targetEmail {
		newActive := ""
		for email := range sessions {
			newActive = email
			break
		}
		cfg.ActiveSession = newActive
		if err := store.SaveConfig(cfg); err != nil {
			return fmt.Errorf("updating config: %w", err)
		}
		if newActive != "" {
			logger.Debug("active session reassigned", "email", newActive)
		}
	}

	fmt.Printf("✓ Logged out %s\n", targetEmail)
	if cfg.ActiveSession != "" && cfg.ActiveSession != targetEmail {
		fmt.Printf("  Active session: %s\n", cfg.ActiveSession)
	} else if cfg.ActiveSession == "" && len(sessions) == 0 {
		fmt.Println("  No sessions remaining.")
	}

	return nil
}
