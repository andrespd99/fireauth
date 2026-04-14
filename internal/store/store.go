package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/cashea-bnpl/auth-devtools/internal/config"
	"github.com/cashea-bnpl/auth-devtools/internal/logger"
)

const (
	configFile   = "config.json"
	sessionsFile = "sessions.json"
	filePerm     = 0600
)

// --- Config ---

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
			return nil, fmt.Errorf("config not found — run 'cashea-auth init' first")
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config.json: %w", err)
	}
	logger.Debug("config loaded", "active_session", cfg.ActiveSession)
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

// --- Sessions ---

// LoadSessions reads the sessions file. Returns an empty map if the file does
// not exist yet.
func LoadSessions() (Sessions, error) {
	path, err := config.FilePath(sessionsFile)
	if err != nil {
		return nil, err
	}
	logger.Debug("loading sessions", "path", path)

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
	logger.Debug("sessions loaded", "count", len(s))
	return s, nil
}

// SaveSessions writes the sessions map to disk.
func SaveSessions(s Sessions) error {
	path, err := config.FilePath(sessionsFile)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	logger.Debug("saving sessions", "path", path, "count", len(s))
	return os.WriteFile(path, data, filePerm)
}

// --- Helpers ---

// GetActiveSession returns the currently active session. It loads both config
// and sessions, validates the active session exists, and returns it.
func GetActiveSession() (*Config, *Session, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, nil, err
	}
	if cfg.ActiveSession == "" {
		return nil, nil, errors.New("no active session — run 'cashea-auth login' first")
	}

	sessions, err := LoadSessions()
	if err != nil {
		return nil, nil, err
	}

	sess, ok := sessions[cfg.ActiveSession]
	if !ok {
		return nil, nil, fmt.Errorf("active session %q not found in sessions — run 'cashea-auth login'", cfg.ActiveSession)
	}

	logger.Debug("active session resolved", "email", sess.Email, "uid", sess.UID)
	return cfg, sess, nil
}

// SetActiveSession updates the active session email in the config file.
func SetActiveSession(email string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	cfg.ActiveSession = email
	logger.Debug("switching active session", "email", email)
	return SaveConfig(cfg)
}

// UpdateSession updates (or adds) a session and saves to disk.
func UpdateSession(sess *Session) error {
	sessions, err := LoadSessions()
	if err != nil {
		return err
	}
	sessions[sess.Email] = sess
	return SaveSessions(sessions)
}
