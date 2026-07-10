package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/andrespd99/fireauth/internal/logger"
	"github.com/andrespd99/fireauth/internal/store"
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
		// Interactive picker.
		emails := make([]string, 0, len(sessions))
		for email := range sessions {
			emails = append(emails, email)
		}
		sort.Strings(emails)

		p, err := store.LoadProject(projectName)
		if err != nil {
			return err
		}

		fmt.Println()
		for i, email := range emails {
			marker := " "
			if email == p.ActiveSession {
				marker = "*"
			}
			fmt.Printf("  %s %d) %s\n", marker, i+1, email)
		}
		fmt.Println()
		fmt.Print("Select session number: ")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}

		num, err := parseInt(strings.TrimSpace(input))
		if err != nil || num < 1 || num > len(emails) {
			return fmt.Errorf("invalid selection")
		}
		targetEmail = emails[num-1]
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