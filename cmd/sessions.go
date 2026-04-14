package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/cashea-bnpl/auth-devtools/internal/store"
	"github.com/spf13/cobra"
)

var flagSessionsJSON bool

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List all stored sessions",
	Long:  "Display all locally stored Firebase Auth sessions with their token status.",
	RunE:  runSessions,
}

func init() {
	sessionsCmd.Flags().BoolVar(&flagSessionsJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(sessionsCmd)
}

func runSessions(cmd *cobra.Command, args []string) error {
	cfg, err := store.LoadConfig()
	if err != nil {
		return err
	}

	sessions, err := store.LoadSessions()
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions stored. Run 'cashea-auth login' to sign in.")
		return nil
	}

	if flagSessionsJSON {
		data, _ := json.MarshalIndent(sessions, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Sort by email for consistent output.
	emails := make([]string, 0, len(sessions))
	for email := range sessions {
		emails = append(emails, email)
	}
	sort.Strings(emails)

	fmt.Println()
	fmt.Printf("  %-30s %-12s %-20s %s\n", "EMAIL", "UID", "TOKEN", "")
	fmt.Printf("  %-30s %-12s %-20s %s\n", "-----", "---", "-----", "")

	for _, email := range emails {
		sess := sessions[email]
		uid := sess.UID
		if len(uid) > 10 {
			uid = uid[:10] + "…"
		}

		tokenStatus := tokenStatusString(sess.TokenExpiry)

		active := ""
		if email == cfg.ActiveSession {
			active = " ← active"
		}

		fmt.Printf("  %-30s %-12s %-20s%s\n", sess.Email, uid, tokenStatus, active)
	}
	fmt.Println()

	return nil
}

func tokenStatusString(expiry time.Time) string {
	remaining := time.Until(expiry)
	if remaining <= 0 {
		return "\033[31mexpired\033[0m"
	}
	if remaining < 5*time.Minute {
		return fmt.Sprintf("\033[33mexpiring (%s)\033[0m", formatDuration(remaining))
	}
	return fmt.Sprintf("\033[32mvalid (%s)\033[0m", formatDuration(remaining))
}
