package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cashea-bnpl/auth-devtools/internal/firebase"
	"github.com/cashea-bnpl/auth-devtools/internal/logger"
	"github.com/cashea-bnpl/auth-devtools/internal/store"
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

  curl -H "Authorization: Bearer $(cashea-auth token)" https://api.example.com`,
	RunE: runToken,
}

func init() {
	tokenCmd.Flags().BoolVar(&flagHeader, "header", false, "print as 'Authorization: Bearer <token>'")
	tokenCmd.Flags().BoolVar(&flagCopy, "copy", false, "copy token to clipboard (macOS pbcopy)")
	tokenCmd.Flags().BoolVar(&flagRefresh, "refresh", false, "force token refresh even if not expired")
	rootCmd.AddCommand(tokenCmd)
}

func runToken(cmd *cobra.Command, args []string) error {
	cfg, sess, err := store.GetActiveSession()
	if err != nil {
		return err
	}

	// Check if refresh is needed.
	refreshWindow := 5 * time.Minute
	needsRefresh := flagRefresh || time.Now().Add(refreshWindow).After(sess.TokenExpiry)

	if needsRefresh {
		if flagRefresh {
			logger.Debug("forced token refresh requested")
		} else {
			logger.Debug("token expired or expiring soon",
				"expiry", sess.TokenExpiry.Format(time.RFC3339),
				"remaining", time.Until(sess.TokenExpiry).String())
		}

		result, err := firebase.RefreshIDToken(cfg.FirebaseAPIKey, sess.RefreshToken)
		if err != nil {
			return fmt.Errorf("refreshing token: %w\nRun 'cashea-auth login' to re-authenticate", err)
		}

		// Update session with new tokens.
		sess.IDToken = result.IDToken
		sess.RefreshToken = result.RefreshToken
		sess.TokenExpiry = firebase.TokenExpiry(result.ExpiresIn)

		if err := store.UpdateSession(sess); err != nil {
			return fmt.Errorf("saving refreshed session: %w", err)
		}
		logger.Debug("token refreshed", "new_expiry", sess.TokenExpiry.Format(time.RFC3339))
	} else {
		logger.Debug("token still valid",
			"remaining", time.Until(sess.TokenExpiry).String())
	}

	token := sess.IDToken

	// Copy to clipboard if requested.
	if flagCopy {
		copyCmd := exec.Command("pbcopy")
		copyCmd.Stdin = strings.NewReader(token)
		if err := copyCmd.Run(); err != nil {
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
