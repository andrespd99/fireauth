package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/andrespd99/fireauth/internal/config"
	"github.com/andrespd99/fireauth/internal/logger"
)

const (
	configFile   = "config.json"
	sessionsFile = "sessions.json"
	projectFile  = "project.json"
	filePerm     = 0600
	dirPerm      = 0700
)

// --- Config (global) ---

// LoadConfig reads and parses the config file.
func LoadConfig() (*Config, error) {
	path, err := config.FilePath(configFile)
	if err != nil {
		return nil, err
	}
	logger.Debug("loading config", "path", path)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found — run 'fireauth init' first")
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config.json: %w", err)
	}
	logger.Debug("config loaded", "active_project", cfg.ActiveProject)
	return &cfg, nil
}

// SaveConfig writes the config to disk.
func SaveConfig(cfg *Config) error {
	path, err := config.FilePath(configFile)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	logger.Debug("saving config", "path", path)
	return os.WriteFile(path, data, filePerm)
}

// --- Project ---

// LoadProject reads the project.json for the named project.
func LoadProject(name string) (*Project, error) {
	path, err := config.ProjectFilePath(name, projectFile)
	if err != nil {
		return nil, err
	}
	logger.Debug("loading project", "name", name, "path", path)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("project %q not found", name)
		}
		return nil, err
	}

	var p Project
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("invalid project.json for %q: %w", name, err)
	}
	logger.Debug("project loaded", "name", p.Name, "active_session", p.ActiveSession)
	return &p, nil
}

// SaveProject writes the project.json to that project's directory, creating
// the directory if needed.
func SaveProject(p *Project) error {
	pdir, err := config.ProjectDir(p.Name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(pdir, dirPerm); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}

	path := filepath.Join(pdir, projectFile)
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	logger.Debug("saving project", "name", p.Name, "path", path)
	return os.WriteFile(path, data, filePerm)
}

// DeleteProject removes a project's directory entirely.
func DeleteProject(name string) error {
	pdir, err := config.ProjectDir(name)
	if err != nil {
		return err
	}
	logger.Debug("deleting project", "name", name, "path", pdir)
	return os.RemoveAll(pdir)
}

// RenameProject renames a project directory and updates its project.json name
// field. If the renamed project was the active one, the global config is
// updated to the new name.
func RenameProject(oldName, newName string) error {
	if oldName == newName {
		return nil
	}

	pdir, err := config.ProjectsDir()
	if err != nil {
		return err
	}

	// Ensure the new name doesn't already exist.
	newPath := filepath.Join(pdir, newName)
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("project %q already exists", newName)
	}

	oldPath := filepath.Join(pdir, oldName)
	logger.Debug("renaming project", "from", oldName, "to", newName, "path", oldPath)

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("renaming project directory: %w", err)
	}

	// Update the name and service account path inside project.json.
	p, err := LoadProject(newName)
	if err != nil {
		return err
	}
	p.Name = newName
	// The service account file lives inside the project directory, so its
	// absolute path changed when the directory was renamed. Rewrite it to
	// point at the new location so downstream commands (e.g. 'me') can find it.
	if p.ServiceAccountPath != "" {
		saFile := filepath.Base(p.ServiceAccountPath)
		p.ServiceAccountPath = filepath.Join(newPath, saFile)
	}
	// SaveProject writes to the new directory using p.Name, which now matches.
	if err := SaveProject(p); err != nil {
		return fmt.Errorf("updating project name: %w", err)
	}

	// Update active project in the global config if needed.
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	if cfg.ActiveProject == oldName {
		cfg.ActiveProject = newName
		if err := SaveConfig(cfg); err != nil {
			return fmt.Errorf("updating active project: %w", err)
		}
	}

	logger.Debug("project renamed", "from", oldName, "to", newName)
	return nil
}

// ListProjects returns the names of all configured projects by scanning the
// projects directory.
func ListProjects() ([]string, error) {
	pdir, err := config.ProjectsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(pdir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			// Only include directories that actually contain a project.json.
			if _, err := os.Stat(filepath.Join(pdir, e.Name(), projectFile)); err == nil {
				names = append(names, e.Name())
			}
		}
	}
	logger.Debug("listed projects", "count", len(names), "names", names)
	return names, nil
}

// GetActiveProjectName returns the active project name from the global config.
func GetActiveProjectName() (string, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return "", err
	}
	if cfg.ActiveProject == "" {
		return "", errors.New("no active project — run 'fireauth project use <name>'")
	}
	return cfg.ActiveProject, nil
}

// SetActiveProject updates the active project name in the global config.
func SetActiveProject(name string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	cfg.ActiveProject = name
	logger.Debug("switching active project", "name", name)
	return SaveConfig(cfg)
}

// --- Sessions (per-project) ---

// projectSessionsPath returns the sessions.json path for a project.
func projectSessionsPath(projectName string) (string, error) {
	return config.ProjectFilePath(projectName, sessionsFile)
}

// LoadSessions reads the sessions file for the given project. Returns an
// empty map if the file does not exist yet.
func LoadSessions(projectName string) (Sessions, error) {
	path, err := projectSessionsPath(projectName)
	if err != nil {
		return nil, err
	}
	logger.Debug("loading sessions", "project", projectName, "path", path)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("no sessions file, returning empty map")
			return make(Sessions), nil
		}
		return nil, err
	}

	var s Sessions
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("invalid sessions.json: %w", err)
	}
	logger.Debug("sessions loaded", "project", projectName, "count", len(s))
	return s, nil
}

// SaveSessions writes the sessions map for the given project to disk.
func SaveSessions(projectName string, s Sessions) error {
	pdir, err := config.ProjectDir(projectName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(pdir, dirPerm); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}

	path, err := projectSessionsPath(projectName)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	logger.Debug("saving sessions", "project", projectName, "path", path, "count", len(s))
	return os.WriteFile(path, data, filePerm)
}

// --- Migration (legacy single-project → multi-project) ---

// MigrateLegacyConfig checks whether the global config.json still uses the old
// single-project schema (FirebaseAPIKey set but no ActiveProject). If so, it
// creates a "default" project from those fields, moves the service account
// file into the project directory, moves sessions.json into the project
// directory, and updates config.json to point to the new "default" project.
//
// It is idempotent: if already migrated or no legacy config exists, it is a
// no-op.
func MigrateLegacyConfig() error {
	cfg, err := LoadConfig()
	if err != nil {
		// No config at all — nothing to migrate.
		return nil
	}
	if cfg.ActiveProject != "" {
		// Already migrated.
		return nil
	}
	if cfg.FirebaseAPIKey == "" {
		// Nothing to migrate from.
		return nil
	}

	logger.Debug("migrating legacy single-project config to 'default' project")

	const defaultProject = "default"

	pdir, err := config.ProjectDir(defaultProject)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(pdir, dirPerm); err != nil {
		return fmt.Errorf("creating default project directory: %w", err)
	}

	// Move service account file if it lives in the root config dir.
	saPath := cfg.ServiceAccountPath
	if saPath != "" {
		rootDir, _ := config.Dir()
		if filepath.Dir(saPath) == rootDir {
			newSA := filepath.Join(pdir, filepath.Base(saPath))
			if err := os.Rename(saPath, newSA); err != nil {
				return fmt.Errorf("moving service account: %w", err)
			}
			saPath = newSA
		}
	}

	// Move sessions.json from root into the project dir.
	oldSessions, err := config.FilePath(sessionsFile)
	if err != nil {
		return err
	}
	newSessions := filepath.Join(pdir, sessionsFile)
	if data, rerr := os.ReadFile(oldSessions); rerr == nil {
		if werr := os.WriteFile(newSessions, data, filePerm); werr != nil {
			return fmt.Errorf("copying sessions.json: %w", werr)
		}
		_ = os.Remove(oldSessions)
	}

	// Create the project.json.
	p := &Project{
		Name:               defaultProject,
		FirebaseAPIKey:     cfg.FirebaseAPIKey,
		ServiceAccountPath: saPath,
		ActiveSession:      cfg.ActiveSession,
	}
	if err := SaveProject(p); err != nil {
		return fmt.Errorf("saving default project: %w", err)
	}

	// Update the global config to point to the default project and clear
	// legacy fields.
	cfg.ActiveProject = defaultProject
	cfg.FirebaseAPIKey = ""
	cfg.ServiceAccountPath = ""
	cfg.ActiveSession = ""
	if err := SaveConfig(cfg); err != nil {
		return fmt.Errorf("updating config after migration: %w", err)
	}

	logger.Debug("legacy config migrated", "project", defaultProject)
	return nil
}

// --- Helpers ---

// GetActiveSession resolves the active project, loads its config, validates
// the active session exists, and returns both the project and session.
func GetActiveSession() (*Project, *Session, error) {
	projectName, err := GetActiveProjectName()
	if err != nil {
		return nil, nil, err
	}
	return GetSession(projectName, "")
}

// GetSession loads a project and a specific session (or the project's active
// session if email is empty).
func GetSession(projectName, email string) (*Project, *Session, error) {
	p, err := LoadProject(projectName)
	if err != nil {
		return nil, nil, err
	}

	if email == "" {
		email = p.ActiveSession
	}
	if email == "" {
		return nil, nil, errors.New("no active session — run 'fireauth login' first")
	}

	sessions, err := LoadSessions(projectName)
	if err != nil {
		return nil, nil, err
	}

	sess, ok := sessions[email]
	if !ok {
		return nil, nil, fmt.Errorf("session %q not found in project %q — run 'fireauth login'", email, projectName)
	}

	logger.Debug("active session resolved", "project", projectName, "email", sess.Email, "uid", sess.UID)
	return p, sess, nil
}

// SetActiveSession updates the active session email in the project config.
func SetActiveSession(projectName, email string) error {
	p, err := LoadProject(projectName)
	if err != nil {
		return err
	}
	p.ActiveSession = email
	logger.Debug("switching active session", "project", projectName, "email", email)
	return SaveProject(p)
}

// UpdateSession updates (or adds) a session for the given project and saves
// to disk.
func UpdateSession(projectName string, sess *Session) error {
	sessions, err := LoadSessions(projectName)
	if err != nil {
		return err
	}
	sessions[sess.Email] = sess
	return SaveSessions(projectName, sessions)
}

// DeleteSession removes a session from a project. If the removed session was
// the active one, the active session is reassigned to any remaining session
// (or cleared).
func DeleteSession(projectName, email string) error {
	sessions, err := LoadSessions(projectName)
	if err != nil {
		return err
	}
	delete(sessions, email)
	if err := SaveSessions(projectName, sessions); err != nil {
		return err
	}

	// Update active session if needed.
	p, err := LoadProject(projectName)
	if err != nil {
		return err
	}
	if p.ActiveSession == email {
		newActive := ""
		for e := range sessions {
			newActive = e
			break
		}
		p.ActiveSession = newActive
		if err := SaveProject(p); err != nil {
			return err
		}
	}
	return nil
}