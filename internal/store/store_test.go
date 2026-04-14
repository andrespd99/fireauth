package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupTestDir overrides the home directory so the config package resolves to a
// temp directory. We set HOME because config.Dir() uses os.UserHomeDir().
func setupTestDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	// Create the .cashea-auth dir inside the temp home.
	dir := filepath.Join(tmp, ".cashea-auth")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSaveAndLoadConfig(t *testing.T) {
	setupTestDir(t)

	cfg := &Config{
		FirebaseAPIKey:     "test-api-key",
		ServiceAccountPath: "/tmp/sa.json",
		ActiveSession:      "user@example.com",
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if loaded.FirebaseAPIKey != cfg.FirebaseAPIKey {
		t.Errorf("FirebaseAPIKey = %q, want %q", loaded.FirebaseAPIKey, cfg.FirebaseAPIKey)
	}
	if loaded.ServiceAccountPath != cfg.ServiceAccountPath {
		t.Errorf("ServiceAccountPath = %q, want %q", loaded.ServiceAccountPath, cfg.ServiceAccountPath)
	}
	if loaded.ActiveSession != cfg.ActiveSession {
		t.Errorf("ActiveSession = %q, want %q", loaded.ActiveSession, cfg.ActiveSession)
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	setupTestDir(t)

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when config.json does not exist")
	}
}

func TestSaveAndLoadSessions(t *testing.T) {
	setupTestDir(t)

	expiry := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	sessions := Sessions{
		"alice@example.com": {
			Email:        "alice@example.com",
			UID:          "uid-alice",
			IDToken:      "token-alice",
			RefreshToken: "refresh-alice",
			TokenExpiry:  expiry,
			DisplayName:  "Alice",
		},
		"bob@example.com": {
			Email:        "bob@example.com",
			UID:          "uid-bob",
			IDToken:      "token-bob",
			RefreshToken: "refresh-bob",
			TokenExpiry:  expiry,
			DisplayName:  "Bob",
		},
	}

	if err := SaveSessions(sessions); err != nil {
		t.Fatalf("SaveSessions: %v", err)
	}

	loaded, err := LoadSessions()
	if err != nil {
		t.Fatalf("LoadSessions: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("len(sessions) = %d, want 2", len(loaded))
	}
	if loaded["alice@example.com"].UID != "uid-alice" {
		t.Error("alice session UID mismatch")
	}
	if loaded["bob@example.com"].DisplayName != "Bob" {
		t.Error("bob session DisplayName mismatch")
	}
}

func TestLoadSessions_Empty(t *testing.T) {
	setupTestDir(t)

	sessions, err := LoadSessions()
	if err != nil {
		t.Fatalf("LoadSessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected empty sessions, got %d", len(sessions))
	}
}

func TestGetActiveSession_NoConfig(t *testing.T) {
	setupTestDir(t)

	_, _, err := GetActiveSession()
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
}

func TestGetActiveSession_NoActiveSet(t *testing.T) {
	setupTestDir(t)

	cfg := &Config{FirebaseAPIKey: "key", ServiceAccountPath: "/tmp/sa.json"}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	_, _, err := GetActiveSession()
	if err == nil {
		t.Fatal("expected error when active_session is empty")
	}
}

func TestSetActiveSession(t *testing.T) {
	setupTestDir(t)

	cfg := &Config{
		FirebaseAPIKey:     "key",
		ServiceAccountPath: "/tmp/sa.json",
		ActiveSession:      "old@example.com",
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	if err := SetActiveSession("new@example.com"); err != nil {
		t.Fatalf("SetActiveSession: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ActiveSession != "new@example.com" {
		t.Errorf("ActiveSession = %q, want %q", loaded.ActiveSession, "new@example.com")
	}
}

func TestFilePermissions(t *testing.T) {
	dir := setupTestDir(t)

	cfg := &Config{FirebaseAPIKey: "key", ServiceAccountPath: "/tmp/sa.json"}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("config.json permissions = %o, want 0600", perm)
	}
}

func TestUpdateSession(t *testing.T) {
	setupTestDir(t)

	sess := &Session{
		Email:        "test@example.com",
		UID:          "uid-test",
		IDToken:      "tok",
		RefreshToken: "ref",
		TokenExpiry:  time.Now().Add(time.Hour),
		DisplayName:  "Test",
	}

	if err := UpdateSession(sess); err != nil {
		t.Fatalf("UpdateSession: %v", err)
	}

	sessions, err := LoadSessions()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sessions["test@example.com"]; !ok {
		t.Error("expected session for test@example.com")
	}

	// Update existing session.
	sess.DisplayName = "Updated"
	if err := UpdateSession(sess); err != nil {
		t.Fatal(err)
	}

	sessions, err = LoadSessions()
	if err != nil {
		t.Fatal(err)
	}
	if sessions["test@example.com"].DisplayName != "Updated" {
		t.Error("expected updated display name")
	}
}

func TestConfigJSON_RoundTrip(t *testing.T) {
	cfg := &Config{
		FirebaseAPIKey:     "AIzaSy_test",
		ServiceAccountPath: "/home/user/.cashea-auth/service-account.json",
		ActiveSession:      "dev@cashea.com",
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var out Config
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out != *cfg {
		t.Errorf("round-trip mismatch: got %+v", out)
	}
}
