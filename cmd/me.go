package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/andrespd99/fireauth/internal/firebase"
	"github.com/andrespd99/fireauth/internal/logger"
	"github.com/andrespd99/fireauth/internal/store"
	"github.com/spf13/cobra"
)

var flagJSON bool

var meCmd = &cobra.Command{
	Use:     "me",
	Aliases: []string{"whoami"},
	Short:   "Show current user details",
	Long:    "Fetch and display the Firebase Auth user record for the active session.",
	RunE:    runMe,
}

func init() {
	meCmd.Flags().BoolVar(&flagJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(meCmd)
}

func runMe(cmd *cobra.Command, args []string) error {
	projectName, err := resolveProjectName()
	if err != nil {
		return err
	}

	ctx := context.Background()
	user, sess, projectName, err := getMe(ctx, projectName)
	if err != nil {
		return err
	}

	// Token status.
	tokenStatus := "valid"
	remaining := time.Until(sess.TokenExpiry)
	if remaining <= 0 {
		tokenStatus = "EXPIRED"
	} else {
		tokenStatus = fmt.Sprintf("valid (%s remaining)", formatDuration(remaining))
	}

	if flagJSON {
		data, err := json.MarshalIndent(buildUserResponse(projectName, user, tokenStatus), "", "  ")
		if err != nil {
			return fmt.Errorf("marshalling user response: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Println()
	fmt.Printf("  Project:        %s\n", projectName)
	fmt.Printf("  Email:          %s\n", user.Email)
	fmt.Printf("  UID:            %s\n", user.UID)
	fmt.Printf("  Display Name:   %s\n", user.DisplayName)
	fmt.Printf("  Email Verified: %v\n", user.EmailVerified)
	fmt.Printf("  Disabled:       %v\n", user.Disabled)
	if len(user.CustomClaims) > 0 {
		claims, _ := json.Marshal(user.CustomClaims)
		fmt.Printf("  Custom Claims:  %s\n", string(claims))
	}
	fmt.Printf("  Created:        %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Last Sign-In:   %s\n", user.LastSignIn.Format("2006-01-02 15:04:05"))
	if len(user.Providers) > 0 {
		fmt.Printf("  Providers:      %s\n", strings.Join(user.Providers, ", "))
	}
	fmt.Printf("  Token:          %s\n", tokenStatus)
	fmt.Println()

	return nil
}

// getMe fetches the Firebase Auth user record for the active session in the
// given project. It returns the user info and the current session.
func getMe(ctx context.Context, projectName string) (*firebase.UserInfo, *store.Session, string, error) {
	p, sess, err := store.GetSession(projectName, "")
	if err != nil {
		return nil, nil, "", err
	}

	logger.Debug("loading admin client", "project", projectName, "service_account", p.ServiceAccountPath)
	admin, err := firebase.NewAdminClient(p.ServiceAccountPath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("initialising admin client: %w", err)
	}

	user, err := admin.GetUserByUID(ctx, sess.UID)
	if err != nil {
		return nil, nil, "", err
	}

	return user, sess, projectName, nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

// UserResponse is the JSON-serialisable representation of a user's details,
// shared between the `me` CLI command and the `/me` serve endpoint.
type UserResponse struct {
	Project       string   `json:"project"`
	UID           string   `json:"uid"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	DisplayName   string   `json:"display_name"`
	Disabled      bool     `json:"disabled"`
	CustomClaims  any      `json:"custom_claims"`
	CreatedAt     string   `json:"created_at"`
	LastSignIn    string   `json:"last_sign_in"`
	Providers     []string `json:"providers"`
	TokenStatus   string   `json:"token_status"`
}

func buildUserResponse(projectName string, user *firebase.UserInfo, tokenStatus string) UserResponse {
	return UserResponse{
		Project:       projectName,
		UID:           user.UID,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		DisplayName:   user.DisplayName,
		Disabled:      user.Disabled,
		CustomClaims:  user.CustomClaims,
		CreatedAt:     user.CreatedAt.Format(time.RFC3339),
		LastSignIn:    user.LastSignIn.Format(time.RFC3339),
		Providers:     user.Providers,
		TokenStatus:   tokenStatus,
	}
}
