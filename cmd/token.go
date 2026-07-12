package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/andrespd99/fireauth/internal/firebase"
	"github.com/andrespd99/fireauth/internal/logger"
	"github.com/andrespd99/fireauth/internal/store"
	"github.com/spf13/cobra"
)

var (
	flagHeader  bool
	flagCopy    bool
	flagRefresh bool
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print the current bearer token",
	Long: `Print the current ID token for the active session.

The token is printed to stdout with no extra formatting, making it easy to use
in shell pipelines:

  curl -H "Authorization: Bearer $(fireauth token)" https://api.example.com`,
	Example: `  fireauth token
  fireauth token -H
  fireauth --project staging token`,
	RunE: runToken,
}

func init() {
	tokenCmd.Flags().BoolVarP(&flagHeader, "header", "H", false, "print as 'Authorization: Bearer <token>'")
	tokenCmd.Flags().BoolVarP(&flagCopy, "copy", "c", false, "copy token to clipboard")
	tokenCmd.Flags().BoolVarP(&flagRefresh, "refresh", "r", false, "force token refresh even if not expired")
	rootCmd.AddCommand(tokenCmd)
}

func runToken(cmd *cobra.Command, args []string) error {
	projectName, err := resolveProjectName()
	if err != nil {
		return err
	}

	token, err := getToken(cmd.Context(), projectName, flagRefresh)
	if err != nil {
		return err
	}

	// Copy to clipboard if requested.
	if flagCopy {
		if err := copyToClipboard(token); err != nil {
			logger.Warn("failed to copy to clipboard", "error", err)
		} else {
			fmt.Fprintln(os.Stderr, "✓ Token copied to clipboard")
		}
	}

	// Print the token.
	if flagHeader {
		fmt.Printf("Authorization: Bearer %s\n", token)
	} else {
		fmt.Print(token)
	}

	return nil
}

// copyToClipboard copies text to the system clipboard using the appropriate
// platform-specific utility.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard utility found (install xclip, xsel, or wl-copy)")
		}
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// getToken returns a valid ID token for the active session in the given project.
// It refreshes the token if expired or within the refresh window (5 minutes),
// or if forceRefresh is true. The refreshed session is persisted to disk.
func getToken(ctx context.Context, projectName string, forceRefresh bool) (string, error) {
	p, sess, err := store.GetSession(projectName, "")
	if err != nil {
		return "", err
	}

	// Check if refresh is needed.
	refreshWindow := 5 * time.Minute
	needsRefresh := forceRefresh || time.Now().Add(refreshWindow).After(sess.TokenExpiry)

	if needsRefresh {
		if forceRefresh {
			logger.Debug("forced token refresh requested")
		} else {
			logger.Debug("token expired or expiring soon",
				"expiry", sess.TokenExpiry.Format(time.RFC3339),
				"remaining", time.Until(sess.TokenExpiry).String())
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := firebase.RefreshIDToken(ctx, p.FirebaseAPIKey, sess.RefreshToken)
		if err != nil {
			return "", fmt.Errorf("refreshing token: %w\nRun 'fireauth login' to re-authenticate", err)
		}

		// Update session with new tokens.
		sess.IDToken = result.IDToken
		sess.RefreshToken = result.RefreshToken
		sess.TokenExpiry = firebase.TokenExpiry(result.ExpiresIn)

		if err := store.UpdateSession(projectName, sess); err != nil {
			return "", fmt.Errorf("saving refreshed session: %w", err)
		}
		logger.Debug("token refreshed", "new_expiry", sess.TokenExpiry.Format(time.RFC3339))
	} else {
		logger.Debug("token still valid",
			"remaining", time.Until(sess.TokenExpiry).String())
	}

	return sess.IDToken, nil
}
