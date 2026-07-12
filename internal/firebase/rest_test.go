package firebase

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSignInWithPassword_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var req SignInRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decoding request: %v", err)
		}
		if req.Email != "test@example.com" {
			t.Errorf("email = %q, want %q", req.Email, "test@example.com")
		}
	if !req.ReturnSecureToken {
		t.Error("expected returnSecureToken to be true")
	}
	if got := r.Header.Get("Referer"); got != "http://localhost" {
		t.Errorf("Referer = %q, want %q", got, "http://localhost")
	}

		resp := SignInResponse{
			IDToken:      "test-id-token-12345",
			RefreshToken: "test-refresh-token",
			ExpiresIn:    "3600",
			LocalID:      "uid-123",
			Email:        "test@example.com",
			DisplayName:  "Test User",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Override the base URL to point at our test server.
	origURL := SignInBaseURL
	SignInBaseURL = server.URL
	defer func() { SignInBaseURL = origURL }()

	result, err := SignInWithPassword("fake-api-key", "test@example.com", "password123")
	if err != nil {
		t.Fatalf("SignInWithPassword: %v", err)
	}
	if result.IDToken != "test-id-token-12345" {
		t.Errorf("IDToken = %q, want %q", result.IDToken, "test-id-token-12345")
	}
	if result.LocalID != "uid-123" {
		t.Errorf("LocalID = %q, want %q", result.LocalID, "uid-123")
	}
}

func TestSignInWithPassword_WrongPassword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		resp := FirebaseError{}
		resp.Error.Code = 400
		resp.Error.Message = "INVALID_PASSWORD"
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	origURL := SignInBaseURL
	SignInBaseURL = server.URL
	defer func() { SignInBaseURL = origURL }()

	_, err := SignInWithPassword("fake-api-key", "test@example.com", "wrong")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
	if got := err.Error(); got != "firebase auth error: invalid password" {
		t.Errorf("error = %q", got)
	}
}

func TestSignInWithPassword_EmailNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		resp := FirebaseError{}
		resp.Error.Code = 400
		resp.Error.Message = "EMAIL_NOT_FOUND"
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	origURL := SignInBaseURL
	SignInBaseURL = server.URL
	defer func() { SignInBaseURL = origURL }()

	_, err := SignInWithPassword("fake-api-key", "nobody@example.com", "pass")
	if err == nil {
		t.Fatal("expected error for email not found")
	}
}

func TestRefreshIDToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.PostForm.Get("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q, want refresh_token", r.PostForm.Get("grant_type"))
		}
	if r.PostForm.Get("refresh_token") != "old-refresh-token" {
		t.Errorf("refresh_token = %q", r.PostForm.Get("refresh_token"))
	}
	if got := r.Header.Get("Referer"); got != "http://localhost" {
		t.Errorf("Referer = %q, want %q", got, "http://localhost")
	}

		resp := RefreshResponse{
			IDToken:      "new-id-token",
			RefreshToken: "new-refresh-token",
			ExpiresIn:    "3600",
			UserID:       "uid-123",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	origURL := RefreshBaseURL
	RefreshBaseURL = server.URL
	defer func() { RefreshBaseURL = origURL }()

	result, err := RefreshIDToken("fake-api-key", "old-refresh-token")
	if err != nil {
		t.Fatalf("RefreshIDToken: %v", err)
	}
	if result.IDToken != "new-id-token" {
		t.Errorf("IDToken = %q, want %q", result.IDToken, "new-id-token")
	}
	if result.RefreshToken != "new-refresh-token" {
		t.Errorf("RefreshToken = %q, want %q", result.RefreshToken, "new-refresh-token")
	}
}

func TestRefreshIDToken_Expired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		resp := FirebaseError{}
		resp.Error.Code = 400
		resp.Error.Message = "TOKEN_EXPIRED"
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	origURL := RefreshBaseURL
	RefreshBaseURL = server.URL
	defer func() { RefreshBaseURL = origURL }()

	_, err := RefreshIDToken("fake-api-key", "expired-token")
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestTokenExpiry(t *testing.T) {
	before := time.Now()
	expiry := TokenExpiry("3600")
	after := time.Now()

	// Should be ~1 hour from now.
	if expiry.Before(before.Add(3599 * time.Second)) {
		t.Error("expiry is too early")
	}
	if expiry.After(after.Add(3601 * time.Second)) {
		t.Error("expiry is too late")
	}
}

func TestTokenExpiry_Invalid(t *testing.T) {
	// Should default to 1 hour for invalid input.
	before := time.Now()
	expiry := TokenExpiry("invalid")
	if expiry.Before(before.Add(3599 * time.Second)) {
		t.Error("expected ~1 hour default for invalid expiresIn")
	}
}

func TestSignInWithPassword_CustomReferer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Referer"); got != "https://myapp.example.com" {
			t.Errorf("Referer = %q, want %q", got, "https://myapp.example.com")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SignInResponse{})
	}))
	defer server.Close()

	origURL := SignInBaseURL
	origReferer := RefererHeader
	SignInBaseURL = server.URL
	RefererHeader = "https://myapp.example.com"
	defer func() {
		SignInBaseURL = origURL
		RefererHeader = origReferer
	}()

	_, err := SignInWithPassword("fake-api-key", "test@example.com", "pass")
	if err != nil {
		t.Fatalf("SignInWithPassword: %v", err)
	}
}

func TestRefreshIDToken_CustomReferer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Referer"); got != "https://myapp.example.com" {
			t.Errorf("Referer = %q, want %q", got, "https://myapp.example.com")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RefreshResponse{})
	}))
	defer server.Close()

	origURL := RefreshBaseURL
	origReferer := RefererHeader
	RefreshBaseURL = server.URL
	RefererHeader = "https://myapp.example.com"
	defer func() {
		RefreshBaseURL = origURL
		RefererHeader = origReferer
	}()

	_, err := RefreshIDToken("fake-api-key", "refresh-token")
	if err != nil {
		t.Fatalf("RefreshIDToken: %v", err)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("abcdefghij", 5); got != "abcde..." {
		t.Errorf("truncate = %q", got)
	}
	if got := truncate("abc", 5); got != "abc" {
		t.Errorf("truncate short = %q", got)
	}
}

func TestFriendlyError(t *testing.T) {
	tests := map[string]string{
		"INVALID_PASSWORD":         "invalid password",
		"EMAIL_NOT_FOUND":          "email not found — check the email address",
		"USER_DISABLED":            "this account has been disabled",
		"INVALID_LOGIN_CREDENTIALS": "invalid email or password",
		"UNKNOWN_ERROR":            "UNKNOWN_ERROR",
	}
	for input, want := range tests {
		if got := friendlyError(input); got != want {
			t.Errorf("friendlyError(%q) = %q, want %q", input, got, want)
		}
	}
}
