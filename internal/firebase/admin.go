package firebase

import (
	"context"
	"fmt"
	"time"

	fb "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/cashea-bnpl/auth-devtools/internal/logger"
	"google.golang.org/api/option"
)

// UserInfo holds the fields we display for a user record.
type UserInfo struct {
	UID           string
	Email         string
	EmailVerified bool
	DisplayName   string
	Disabled      bool
	CustomClaims  map[string]interface{}
	CreatedAt     time.Time
	LastSignIn    time.Time
	Providers     []string
}

// UserGetter is an interface for fetching user info. It enables unit testing
// without a real Firebase project.
type UserGetter interface {
	GetUser(ctx context.Context, uid string) (*auth.UserRecord, error)
}

// AdminClient wraps the Firebase Admin Auth client.
type AdminClient struct {
	auth UserGetter
}

// NewAdminClient initialises the Firebase Admin SDK from a service account file.
func NewAdminClient(serviceAccountPath string) (*AdminClient, error) {
	logger.Debug("initialising firebase admin SDK", "service_account", serviceAccountPath)

	ctx := context.Background()
	opt := option.WithCredentialsFile(serviceAccountPath)
	app, err := fb.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("initialising firebase app: %w", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialising firebase auth client: %w", err)
	}

	return &AdminClient{auth: authClient}, nil
}

// NewAdminClientWithGetter creates an AdminClient with a custom UserGetter
// (used for testing).
func NewAdminClientWithGetter(getter UserGetter) *AdminClient {
	return &AdminClient{auth: getter}
}

// GetUserByUID fetches a user record from Firebase Auth by UID.
func (c *AdminClient) GetUserByUID(ctx context.Context, uid string) (*UserInfo, error) {
	logger.Debug("fetching user by UID", "uid", uid)

	record, err := c.auth.GetUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("getting user %s: %w", uid, err)
	}

	info := &UserInfo{
		UID:           record.UID,
		Email:         record.Email,
		EmailVerified: record.EmailVerified,
		DisplayName:   record.DisplayName,
		Disabled:      record.Disabled,
		CustomClaims:  record.CustomClaims,
		CreatedAt:     time.UnixMilli(record.UserMetadata.CreationTimestamp),
		LastSignIn:    time.UnixMilli(record.UserMetadata.LastLogInTimestamp),
	}

	for _, p := range record.ProviderUserInfo {
		info.Providers = append(info.Providers, p.ProviderID)
	}

	logger.Debug("user fetched", "uid", info.UID, "email", info.Email, "providers", info.Providers)
	return info, nil
}
