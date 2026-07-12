package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/andrespd99/fireauth/internal/firebase"
	"github.com/andrespd99/fireauth/internal/logger"
	"github.com/andrespd99/fireauth/internal/store"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	flagEmail    string
	flagPassword string
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in with email and password",
	Long:  "Authenticate against Firebase Auth and store the session locally.",
	RunE:  runLogin,
}

func init() {
	loginCmd.Flags().StringVar(&flagEmail, "email", "", "user email (skips prompt)")
	loginCmd.Flags().StringVar(&flagPassword, "password", "", "user password (skips prompt — warning: visible in shell history)")
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	projectName, err := resolveProjectName()
	if err != nil {
		return err
	}

	p, err := store.LoadProject(projectName)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)

	// Get email.
	email := flagEmail
	if email == "" {
		fmt.Print("Email: ")
		email, err = reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading email: %w", err)
		}
		email = strings.TrimSpace(email)
	}
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	// Get password.
	password := flagPassword
	if password == "" {
		fmt.Print("Password: ")
		pwBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}
		fmt.Println() // newline after hidden input
		password = string(pwBytes)
	}
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	logger.Debug("attempting sign-in", "project", projectName, "email", email)

	// Sign in via Firebase REST API.
	result, err := firebase.SignInWithPassword(cmd.Context(), p.FirebaseAPIKey, email, password)
	if err != nil {
		return err
	}

	// Store the session.
	sess := &store.Session{
		Email:        result.Email,
		UID:          result.LocalID,
		IDToken:      result.IDToken,
		RefreshToken: result.RefreshToken,
		TokenExpiry:  firebase.TokenExpiry(result.ExpiresIn),
		DisplayName:  result.DisplayName,
	}
	if err := store.UpdateSession(projectName, sess); err != nil {
		return fmt.Errorf("saving session: %w", err)
	}

	// Set as active session.
	if err := store.SetActiveSession(projectName, sess.Email); err != nil {
		return fmt.Errorf("setting active session: %w", err)
	}

	fmt.Println()
	fmt.Printf("✓ Logged in as %s\n", sess.Email)
	if sess.DisplayName != "" {
		fmt.Printf("  Name: %s\n", sess.DisplayName)
	}
	fmt.Printf("  UID:  %s\n", sess.UID)
	fmt.Printf("  Project: %s\n", projectName)
	fmt.Printf("  Token expires at %s\n", sess.TokenExpiry.Format("15:04:05"))
	return nil
}
