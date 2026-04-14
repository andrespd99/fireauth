package store

import "time"

// Config holds the tool's global configuration persisted in config.json.
type Config struct {
	FirebaseAPIKey     string `json:"firebase_api_key"`
	ServiceAccountPath string `json:"service_account_path"`
	ActiveSession      string `json:"active_session"`
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
