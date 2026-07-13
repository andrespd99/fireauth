package store

import "time"

// Config holds the tool's global configuration persisted in config.json.
// In the multi-project model this only tracks the active project name.
// The legacy single-project fields (FirebaseAPIKey, ServiceAccountPath,
// ActiveSession) are kept for migration and back-compat — they are read into
// the auto-created "default" project on first run.
type Config struct {
	// ActiveProject is the name of the currently selected project.
	ActiveProject string `json:"active_project"`

	// --- Legacy fields (pre-multi-project) ---
	FirebaseAPIKey     string `json:"firebase_api_key,omitempty"`
	ServiceAccountPath string `json:"service_account_path,omitempty"`
	ActiveSession      string `json:"active_session,omitempty"`
}

// Project holds the configuration for a single Firebase project, persisted
// in ~/.fireauth/projects/<name>/project.json.
type Project struct {
	Name               string `json:"name"`
	FirebaseAPIKey     string `json:"firebase_api_key"`
	ServiceAccountPath string `json:"service_account_path"`
	ActiveSession      string `json:"active_session"`
	Referer            string `json:"referer,omitempty"`
}

// Session represents a single authenticated user session.
type Session struct {
	Email        string    `json:"email"`
	UID          string    `json:"uid"`
	IDToken      string    `json:"id_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenExpiry  time.Time `json:"token_expiry"`
	DisplayName  string    `json:"display_name"`
}

// Sessions is a map of email → Session.
type Sessions map[string]*Session