package firebase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andrespd99/fireauth/internal/logger"
)

// Default Firebase Auth REST API base URLs. Overridable for testing.
var (
	SignInBaseURL  = "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword"
	RefreshBaseURL = "https://securetoken.googleapis.com/v1/token"
)

// --- Sign-in ---

// SignInRequest is the payload sent to the Firebase sign-in endpoint.
type SignInRequest struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

// SignInResponse contains the fields we need from a successful sign-in.
type SignInResponse struct {
	IDToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"` // seconds as string
	LocalID      string `json:"localId"`   // UID
	Email        string `json:"email"`
	DisplayName  string `json:"displayName"`
}

// FirebaseError is the error structure returned by the Firebase REST API.
type FirebaseError struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// SignInWithPassword authenticates a user via email/password and returns the
// sign-in response including ID and refresh tokens.
func SignInWithPassword(apiKey, email, password string) (*SignInResponse, error) {
	reqURL := fmt.Sprintf("%s?key=%s", SignInBaseURL, url.QueryEscape(apiKey))
	logger.Debug("firebase sign-in request", "url", SignInBaseURL, "email", email)

	payload := SignInRequest{
		Email:             email,
		Password:          password,
		ReturnSecureToken: true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling sign-in request: %w", err)
	}

	resp, err := http.Post(reqURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("sign-in HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading sign-in response: %w", err)
	}
	logger.Debug("firebase sign-in response", "status", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		var fbErr FirebaseError
		if err := json.Unmarshal(respBody, &fbErr); err == nil && fbErr.Error.Message != "" {
			return nil, fmt.Errorf("firebase auth error: %s", friendlyError(fbErr.Error.Message))
		}
		return nil, fmt.Errorf("firebase auth error: HTTP %d", resp.StatusCode)
	}

	var result SignInResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing sign-in response: %w", err)
	}

	logger.Debug("sign-in successful", "uid", result.LocalID, "token_prefix", truncate(result.IDToken, 10))
	return &result, nil
}

// TokenExpiry computes the absolute expiry time from the ExpiresIn field.
func TokenExpiry(expiresIn string) time.Time {
	secs, err := strconv.Atoi(expiresIn)
	if err != nil {
		secs = 3600 // default to 1 hour
	}
	return time.Now().Add(time.Duration(secs) * time.Second)
}

// --- Token Refresh ---

// RefreshResponse contains the fields from a successful token refresh.
type RefreshResponse struct {
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    string `json:"expires_in"` // seconds as string
	UserID       string `json:"user_id"`
}

// RefreshIDToken exchanges a refresh token for a new ID token.
func RefreshIDToken(apiKey, refreshToken string) (*RefreshResponse, error) {
	reqURL := fmt.Sprintf("%s?key=%s", RefreshBaseURL, url.QueryEscape(apiKey))
	logger.Debug("firebase token refresh request", "url", RefreshBaseURL)

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	resp, err := http.Post(reqURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("refresh HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading refresh response: %w", err)
	}
	logger.Debug("firebase refresh response", "status", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		var fbErr FirebaseError
		if err := json.Unmarshal(respBody, &fbErr); err == nil && fbErr.Error.Message != "" {
			return nil, fmt.Errorf("token refresh error: %s", friendlyError(fbErr.Error.Message))
		}
		return nil, fmt.Errorf("token refresh error: HTTP %d", resp.StatusCode)
	}

	var result RefreshResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing refresh response: %w", err)
	}

	logger.Debug("token refresh successful", "user_id", result.UserID, "token_prefix", truncate(result.IDToken, 10))
	return &result, nil
}

// --- Helpers ---

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func friendlyError(msg string) string {
	switch msg {
	case "EMAIL_NOT_FOUND":
		return "email not found — check the email address"
	case "INVALID_PASSWORD":
		return "invalid password"
	case "USER_DISABLED":
		return "this account has been disabled"
	case "INVALID_LOGIN_CREDENTIALS":
		return "invalid email or password"
	case "TOKEN_EXPIRED":
		return "session expired — run 'fireauth login' to re-authenticate"
	case "INVALID_REFRESH_TOKEN":
		return "refresh token is invalid — run 'fireauth login' to re-authenticate"
	default:
		return msg
	}
}
