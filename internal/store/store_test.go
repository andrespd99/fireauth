package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupTestDir overrides the home directory so the config package resolves to a
// temp directory. We set HOME because config.Dir() uses os.UserHomeDir().
func setupTestDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	// Create the .fireauth dir inside the temp home.
	dir := filepath.Join(tmp, ".fireauth")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

// setupTestProject creates a project with sessions for testing.
func setupTestProject(t *testing.T, name string) string {
	t.Helper()
	setupTestDir(t)

	p := &Project{
		Name:               name,
		FirebaseAPIKey:     "test-api-key",
		ServiceAccountPath: "/tmp/sa.json",
	}
	if err := SaveProject(p); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}

	cfg := &Config{ActiveProject: name}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	return name
}

func TestSaveAndLoadConfig(t *testing.T) {
	setupTestDir(t)

	cfg := &Config{
		ActiveProject: "staging",
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if loaded.ActiveProject != cfg.ActiveProject {
		t.Errorf("ActiveProject = %q, want %q", loaded.ActiveProject, cfg.ActiveProject)
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	setupTestDir(t)

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when config.json does not exist")
	}
}

func TestSaveAndLoadProject(t *testing.T) {
	setupTestDir(t)

	p := &Project{
		Name:               "staging",
		FirebaseAPIKey:     "test-api-key",
		ServiceAccountPath: "/tmp/sa.json",
		ActiveSession:      "user@example.com",
	}

	if err := SaveProject(p); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}

	loaded, err := LoadProject("staging")
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	if loaded.FirebaseAPIKey != p.FirebaseAPIKey {
		t.Errorf("FirebaseAPIKey = %q, want %q", loaded.FirebaseAPIKey, p.FirebaseAPIKey)
	}
	if loaded.ServiceAccountPath != p.ServiceAccountPath {
		t.Errorf("ServiceAccountPath = %q, want %q", loaded.ServiceAccountPath, p.ServiceAccountPath)
	}
	if loaded.ActiveSession != p.ActiveSession {
		t.Errorf("ActiveSession = %q, want %q", loaded.ActiveSession, p.ActiveSession)
	}
}

func TestLoadProject_NotFound(t *testing.T) {
	setupTestDir(t)

	_, err := LoadProject("nonexistent")
	if err == nil {
		t.Fatal("expected error when project does not exist")
	}
}

func TestListProjects(t *testing.T) {
	setupTestDir(t)

	p1 := &Project{Name: "staging", FirebaseAPIKey: "key1"}
	p2 := &Project{Name: "production", FirebaseAPIKey: "key2"}
	if err := SaveProject(p1); err != nil {
		t.Fatal(err)
	}
	if err := SaveProject(p2); err != nil {
		t.Fatal(err)
	}

	projects, err := ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
}

func TestListProjects_Empty(t *testing.T) {
	setupTestDir(t)

	projects, err := ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestDeleteProject(t *testing.T) {
	setupTestDir(t)

	p := &Project{Name: "staging", FirebaseAPIKey: "key"}
	if err := SaveProject(p); err != nil {
		t.Fatal(err)
	}

	if err := DeleteProject("staging"); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}

	_, err := LoadProject("staging")
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}

func TestRenameProject(t *testing.T) {
	setupTestDir(t)

	// Use a real service account file inside the project dir so we can verify
	// the path is rewritten and still exists after the rename.
	pdir, err := filepathGlob("staging")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pdir, 0700); err != nil {
		t.Fatal(err)
	}
	saPath := filepath.Join(pdir, "service-account.json")
	if err := os.WriteFile(saPath, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	p := &Project{Name: "staging", FirebaseAPIKey: "key", ServiceAccountPath: saPath, ActiveSession: "user@example.com"}
	if err := SaveProject(p); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{ActiveProject: "staging"}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Add a session so we can verify it survives the rename.
	sess := &Session{Email: "user@example.com", UID: "uid1"}
	if err := UpdateSession("staging", sess); err != nil {
		t.Fatal(err)
	}

	if err := RenameProject("staging", "production"); err != nil {
		t.Fatalf("RenameProject: %v", err)
	}

	// Old name should be gone.
	if _, err := LoadProject("staging"); err == nil {
		t.Fatal("expected error loading old project name")
	}

	// New name should have the data.
	loaded, err := LoadProject("production")
	if err != nil {
		t.Fatalf("LoadProject(production): %v", err)
	}
	if loaded.FirebaseAPIKey != "key" {
		t.Errorf("FirebaseAPIKey = %q, want %q", loaded.FirebaseAPIKey, "key")
	}
	if loaded.Name != "production" {
		t.Errorf("Name = %q, want %q", loaded.Name, "production")
	}
	if loaded.ActiveSession != "user@example.com" {
		t.Errorf("ActiveSession = %q, want %q", loaded.ActiveSession, "user@example.com")
	}

	// The service account path should be rewritten to the new project dir and
	// the file should still exist at the new location.
	if loaded.ServiceAccountPath == saPath {
		t.Errorf("ServiceAccountPath was not rewritten, still %q", loaded.ServiceAccountPath)
	}
	if !strings.Contains(loaded.ServiceAccountPath, filepath.Join("projects", "production")) {
		t.Errorf("ServiceAccountPath = %q, expected it to point inside the 'production' project dir", loaded.ServiceAccountPath)
	}
	if _, err := os.Stat(loaded.ServiceAccountPath); err != nil {
		t.Errorf("service account file not found at rewritten path %q: %v", loaded.ServiceAccountPath, err)
	}

	// Sessions should survive the rename.
	sessions, err := LoadSessions("production")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sessions["user@example.com"]; !ok {
		t.Error("expected session to survive rename")
	}

	// Active project in global config should be updated.
	cfg, err = LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ActiveProject != "production" {
		t.Errorf("ActiveProject = %q, want %q", cfg.ActiveProject, "production")
	}
}

func TestRenameProject_AlreadyExists(t *testing.T) {
	setupTestDir(t)

	if err := SaveProject(&Project{Name: "staging", FirebaseAPIKey: "key1"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveProject(&Project{Name: "production", FirebaseAPIKey: "key2"}); err != nil {
		t.Fatal(err)
	}

	err := RenameProject("staging", "production")
	if err == nil {
		t.Fatal("expected error renaming to an existing project name")
	}
}

func TestRenameProject_SameName(t *testing.T) {
	setupTestDir(t)

	if err := SaveProject(&Project{Name: "staging", FirebaseAPIKey: "key"}); err != nil {
		t.Fatal(err)
	}

	if err := RenameProject("staging", "staging"); err != nil {
		t.Fatalf("RenameProject same name should be no-op: %v", err)
	}
}

func TestSaveAndLoadSessions(t *testing.T) {
	projectName := setupTestProject(t, "staging")

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

	if err := SaveSessions(projectName, sessions); err != nil {
		t.Fatalf("SaveSessions: %v", err)
	}

	loaded, err := LoadSessions(projectName)
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
	projectName := setupTestProject(t, "staging")

	sessions, err := LoadSessions(projectName)
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

func TestGetActiveSession_NoActiveProject(t *testing.T) {
	setupTestDir(t)

	cfg := &Config{}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	_, _, err := GetActiveSession()
	if err == nil {
		t.Fatal("expected error when active_project is empty")
	}
}

func TestGetActiveSession_NoActiveSet(t *testing.T) {
	projectName := setupTestProject(t, "staging")

	_, _, err := GetSession(projectName, "")
	if err == nil {
		t.Fatal("expected error when active_session is empty")
	}
}

func TestSetActiveSession(t *testing.T) {
	projectName := setupTestProject(t, "staging")

	if err := SetActiveSession(projectName, "new@example.com"); err != nil {
		t.Fatalf("SetActiveSession: %v", err)
	}

	loaded, err := LoadProject(projectName)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ActiveSession != "new@example.com" {
		t.Errorf("ActiveSession = %q, want %q", loaded.ActiveSession, "new@example.com")
	}
}

func TestSetActiveProject(t *testing.T) {
	setupTestDir(t)

	p1 := &Project{Name: "staging", FirebaseAPIKey: "key1"}
	p2 := &Project{Name: "production", FirebaseAPIKey: "key2"}
	if err := SaveProject(p1); err != nil {
		t.Fatal(err)
	}
	if err := SaveProject(p2); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{ActiveProject: "staging"}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	if err := SetActiveProject("production"); err != nil {
		t.Fatalf("SetActiveProject: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ActiveProject != "production" {
		t.Errorf("ActiveProject = %q, want %q", loaded.ActiveProject, "production")
	}
}

func TestFilePermissions(t *testing.T) {
	dir := setupTestDir(t)

	cfg := &Config{ActiveProject: "staging"}
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

func TestProjectFilePermissions(t *testing.T) {
	setupTestDir(t)

	p := &Project{Name: "staging", FirebaseAPIKey: "key"}
	if err := SaveProject(p); err != nil {
		t.Fatal(err)
	}

	pdir, _ := filepathGlob("staging")
	info, err := os.Stat(filepath.Join(pdir, "project.json"))
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("project.json permissions = %o, want 0600", perm)
	}
}

func TestUpdateSession(t *testing.T) {
	projectName := setupTestProject(t, "staging")

	sess := &Session{
		Email:        "test@example.com",
		UID:          "uid-test",
		IDToken:      "tok",
		RefreshToken: "ref",
		TokenExpiry:  time.Now().Add(time.Hour),
		DisplayName:  "Test",
	}

	if err := UpdateSession(projectName, sess); err != nil {
		t.Fatalf("UpdateSession: %v", err)
	}

	sessions, err := LoadSessions(projectName)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sessions["test@example.com"]; !ok {
		t.Error("expected session for test@example.com")
	}

	// Update existing session.
	sess.DisplayName = "Updated"
	if err := UpdateSession(projectName, sess); err != nil {
		t.Fatal(err)
	}

	sessions, err = LoadSessions(projectName)
	if err != nil {
		t.Fatal(err)
	}
	if sessions["test@example.com"].DisplayName != "Updated" {
		t.Error("expected updated display name")
	}
}

func TestDeleteSession(t *testing.T) {
	projectName := setupTestProject(t, "staging")

	sess1 := &Session{Email: "a@example.com", UID: "uid1"}
	sess2 := &Session{Email: "b@example.com", UID: "uid2"}
	if err := UpdateSession(projectName, sess1); err != nil {
		t.Fatal(err)
	}
	if err := UpdateSession(projectName, sess2); err != nil {
		t.Fatal(err)
	}

	// Set active session to sess1.
	if err := SetActiveSession(projectName, "a@example.com"); err != nil {
		t.Fatal(err)
	}

	// Delete active session.
	if err := DeleteSession(projectName, "a@example.com"); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	sessions, err := LoadSessions(projectName)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sessions["a@example.com"]; ok {
		t.Error("session a@example.com should have been deleted")
	}

	// Active session should have been reassigned.
	p, err := LoadProject(projectName)
	if err != nil {
		t.Fatal(err)
	}
	if p.ActiveSession != "b@example.com" {
		t.Errorf("expected active session reassigned to b@example.com, got %q", p.ActiveSession)
	}
}

func TestDeleteSession_LastOne(t *testing.T) {
	projectName := setupTestProject(t, "staging")

	sess := &Session{Email: "only@example.com", UID: "uid1"}
	if err := UpdateSession(projectName, sess); err != nil {
		t.Fatal(err)
	}
	if err := SetActiveSession(projectName, "only@example.com"); err != nil {
		t.Fatal(err)
	}

	if err := DeleteSession(projectName, "only@example.com"); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	p, err := LoadProject(projectName)
	if err != nil {
		t.Fatal(err)
	}
	if p.ActiveSession != "" {
		t.Errorf("expected empty active session, got %q", p.ActiveSession)
	}
}

func TestConfigJSON_RoundTrip(t *testing.T) {
	cfg := &Config{
		ActiveProject: "production",
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var out Config
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.ActiveProject != cfg.ActiveProject {
		t.Errorf("round-trip mismatch: got %+v", out)
	}
}

func TestProjectJSON_RoundTrip(t *testing.T) {
	p := &Project{
		Name:               "production",
		FirebaseAPIKey:     "AIzaSy_test",
		ServiceAccountPath: "/home/user/.fireauthojects/production/service-account.json",
		ActiveSession:      "dev@example.com",
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var out Project
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out != *p {
		t.Errorf("round-trip mismatch: got %+v", out)
	}
}

func TestMigrateLegacyConfig(t *testing.T) {
	dir := setupTestDir(t)

	// Create legacy config.json.
	legacyCfg := &Config{
		FirebaseAPIKey:     "AIzaSy_legacy",
		ServiceAccountPath: filepath.Join(dir, "service-account.json"),
		ActiveSession:      "user@example.com",
	}
	if err := SaveConfig(legacyCfg); err != nil {
		t.Fatal(err)
	}

	// Create the service account file in the root dir.
	if err := os.WriteFile(filepath.Join(dir, "service-account.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	// Create legacy sessions.json in the root dir.
	sessions := Sessions{
		"user@example.com": {
			Email:        "user@example.com",
			UID:          "uid1",
			IDToken:      "token",
			RefreshToken: "refresh",
			TokenExpiry:  time.Now().Add(time.Hour),
		},
	}
	sessionsData, _ := json.MarshalIndent(sessions, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "sessions.json"), sessionsData, 0600); err != nil {
		t.Fatal(err)
	}

	// Run migration.
	if err := MigrateLegacyConfig(); err != nil {
		t.Fatalf("MigrateLegacyConfig: %v", err)
	}

	// Verify global config now has ActiveProject = "default".
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ActiveProject != "default" {
		t.Fatalf("expected ActiveProject 'default', got %q", cfg.ActiveProject)
	}

	// Verify legacy fields are cleared.
	if cfg.FirebaseAPIKey != "" {
		t.Errorf("expected legacy FirebaseAPIKey cleared, got %q", cfg.FirebaseAPIKey)
	}

	// Verify default project exists with the legacy data.
	p, err := LoadProject("default")
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if p.FirebaseAPIKey != "AIzaSy_legacy" {
		t.Errorf("expected API key migrated, got %q", p.FirebaseAPIKey)
	}
	if p.ActiveSession != "user@example.com" {
		t.Errorf("expected active session migrated, got %q", p.ActiveSession)
	}

	// Verify service account file was moved into project dir.
	if _, err := os.Stat(p.ServiceAccountPath); err != nil {
		t.Errorf("service account not found at migrated path %q: %v", p.ServiceAccountPath, err)
	}
	// And removed from root.
	if _, err := os.Stat(filepath.Join(dir, "service-account.json")); !os.IsNotExist(err) {
		t.Error("expected old service account file to be removed from root dir")
	}

	// Verify sessions were migrated.
	migratedSessions, err := LoadSessions("default")
	if err != nil {
		t.Fatal(err)
	}
	if len(migratedSessions) != 1 {
		t.Fatalf("expected 1 migrated session, got %d", len(migratedSessions))
	}
	if _, ok := migratedSessions["user@example.com"]; !ok {
		t.Error("expected user@example.com session to be migrated")
	}

	// Verify old sessions.json is removed from root.
	if _, err := os.Stat(filepath.Join(dir, "sessions.json")); !os.IsNotExist(err) {
		t.Error("expected old sessions.json to be removed from root dir")
	}
}

func TestMigrateLegacyConfig_AlreadyMigrated(t *testing.T) {
	setupTestDir(t)

	cfg := &Config{ActiveProject: "staging"}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Should be a no-op.
	if err := MigrateLegacyConfig(); err != nil {
		t.Fatalf("MigrateLegacyConfig: %v", err)
	}

	// Config should be unchanged.
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ActiveProject != "staging" {
		t.Errorf("ActiveProject changed to %q", loaded.ActiveProject)
	}
}

func TestMigrateLegacyConfig_NoConfig(t *testing.T) {
	setupTestDir(t)

	// Should be a no-op (no error).
	if err := MigrateLegacyConfig(); err != nil {
		t.Fatalf("MigrateLegacyConfig: %v", err)
	}
}

// filepathGlob is a helper to get the project directory path in tests.
func filepathGlob(projectName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".fireauth", "projects", projectName), nil
}
