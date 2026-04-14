package firebase

import (
	"context"
	"errors"
	"testing"

	"firebase.google.com/go/v4/auth"
)

// mockUserGetter implements UserGetter for testing.
type mockUserGetter struct {
	users map[string]*auth.UserRecord
}

func (m *mockUserGetter) GetUser(ctx context.Context, uid string) (*auth.UserRecord, error) {
	u, ok := m.users[uid]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func TestGetUserByUID_Success(t *testing.T) {
	mock := &mockUserGetter{
		users: map[string]*auth.UserRecord{
			"uid-123": {
				UserInfo: &auth.UserInfo{
					UID:         "uid-123",
					Email:       "test@example.com",
					DisplayName: "Test User",
					ProviderID:  "firebase",
				},
				EmailVerified: true,
				Disabled:      false,
				CustomClaims:  map[string]interface{}{"role": "admin"},
				UserMetadata: &auth.UserMetadata{
					CreationTimestamp:  1700000000000,
					LastLogInTimestamp: 1700100000000,
				},
				ProviderUserInfo: []*auth.UserInfo{
					{ProviderID: "password"},
					{ProviderID: "google.com"},
				},
			},
		},
	}

	client := NewAdminClientWithGetter(mock)
	info, err := client.GetUserByUID(context.Background(), "uid-123")
	if err != nil {
		t.Fatalf("GetUserByUID: %v", err)
	}

	if info.UID != "uid-123" {
		t.Errorf("UID = %q, want %q", info.UID, "uid-123")
	}
	if info.Email != "test@example.com" {
		t.Errorf("Email = %q", info.Email)
	}
	if !info.EmailVerified {
		t.Error("expected EmailVerified to be true")
	}
	if info.DisplayName != "Test User" {
		t.Errorf("DisplayName = %q", info.DisplayName)
	}
	if info.Disabled {
		t.Error("expected Disabled to be false")
	}
	if info.CustomClaims["role"] != "admin" {
		t.Errorf("CustomClaims = %v", info.CustomClaims)
	}
	if len(info.Providers) != 2 {
		t.Errorf("Providers = %v, want 2 entries", info.Providers)
	}
}

func TestGetUserByUID_NotFound(t *testing.T) {
	mock := &mockUserGetter{users: map[string]*auth.UserRecord{}}
	client := NewAdminClientWithGetter(mock)

	_, err := client.GetUserByUID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}
