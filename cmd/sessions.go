package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/andrespd99/fireauth/internal/store"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var flagSessionsJSON bool

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List all stored sessions",
	Long:  "Display all locally stored Firebase Auth sessions with their token status.",
	Example: `  fireauth sessions
  fireauth sessions --json`,
	RunE: runSessions,
}

func init() {
	sessionsCmd.Flags().BoolVarP(&flagSessionsJSON, "json", "j", false, "output as JSON")
	rootCmd.AddCommand(sessionsCmd)
}

func runSessions(cmd *cobra.Command, args []string) error {
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

	if len(sessions) == 0 {
		fmt.Printf("No sessions stored for project %q. Run 'fireauth login' to sign in.\n", projectName)
		return nil
	}

	if flagSessionsJSON {
		data, err := json.MarshalIndent(sessions, "", "  ")
		if err != nil {
			return fmt.Errorf("marshalling sessions: %w", err)
		}
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
	fmt.Printf("  Project: %s\n\n", projectName)
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
		if email == p.ActiveSession {
			active = " ← active"
		}

		fmt.Printf("  %-30s %-12s %-20s%s\n", sess.Email, uid, tokenStatus, active)
	}
	fmt.Println()

	return nil
}

var (
	tokenExpiredStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	tokenExpiringStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	tokenValidStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

func tokenStatusString(expiry time.Time) string {
	remaining := time.Until(expiry)
	if remaining <= 0 {
		return tokenExpiredStyle.Render("expired")
	}
	if remaining < 5*time.Minute {
		return tokenExpiringStyle.Render(fmt.Sprintf("expiring (%s)", formatDuration(remaining)))
	}
	return tokenValidStyle.Render(fmt.Sprintf("valid (%s)", formatDuration(remaining)))
}
