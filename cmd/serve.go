package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/andrespd99/fireauth/internal/logger"
	"github.com/andrespd99/fireauth/internal/store"
	"github.com/spf13/cobra"
)

var flagAddr string

const defaultAddr = "127.0.0.1:9876"

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start a local HTTP server for Postman integration",
	Long: `Start a local HTTP server that exposes endpoints for retrieving tokens
and user data. Designed for use with Postman pre-request scripts.

Binds to 127.0.0.1 only (no remote access). All endpoints accept an optional
?project= query parameter to override the active project.

Endpoints:

  GET /health        Health check (returns version + status)
  GET /token         Returns the bearer token for the active session
  GET /me            Returns JSON user details for the active session

Query parameters for /token:
  project   Override the active project for this request
  refresh   Force token refresh (true/false, default false)
  format    "header" to get "Authorization: Bearer <token>", otherwise bare token

Query parameters for /me:
  project   Override the active project for this request`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().StringVar(&flagAddr, "addr", defaultAddr, "address to listen on")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	mux := http.NewServeMux()

	srv := &server{}

	mux.HandleFunc("GET /health", srv.handleHealth)
	mux.HandleFunc("GET /token", srv.handleToken)
	mux.HandleFunc("GET /me", srv.handleMe)

	logger.Info("starting server", "addr", flagAddr)
	fmt.Fprintf(cmd.ErrOrStderr(), "fireauth server listening on http://%s\n", flagAddr)
	fmt.Fprintf(cmd.ErrOrStderr(), "Press Ctrl+C to stop\n")

	httpServer := &http.Server{
		Addr:    flagAddr,
		Handler: mux,
	}

	return httpServer.ListenAndServe()
}

type server struct {
	// sessionMu guards against concurrent session file writes (e.g. when
	// multiple /token requests trigger a refresh at the same time).
	sessionMu sync.Mutex
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// resolveProjectParam resolves the project from the ?project= query parameter,
// falling back to the active project in config.
func resolveProjectParam(r *http.Request) (string, error) {
	if p := r.URL.Query().Get("project"); p != "" {
		return p, nil
	}
	return store.GetActiveProjectName()
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": version,
	})
}

func (s *server) handleToken(w http.ResponseWriter, r *http.Request) {
	projectName, err := resolveProjectParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	forceRefresh := r.URL.Query().Get("refresh") == "true"
	format := r.URL.Query().Get("format")

	// Guard session writes against concurrent requests.
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()

	token, err := getToken(projectName, forceRefresh)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if format == "header" {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Authorization: Bearer %s", token)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, token)
}

func (s *server) handleMe(w http.ResponseWriter, r *http.Request) {
	projectName, err := resolveProjectParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	user, sess, projName, err := getMe(ctx, projectName)
	if err != nil {
		// Distinguish "no session" errors from backend errors.
		msg := err.Error()
		if strings.Contains(msg, "no active") || strings.Contains(msg, "run 'fireauth init'") {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		writeError(w, http.StatusInternalServerError, msg)
		return
	}

	tokenStatus := "valid"
	remaining := time.Until(sess.TokenExpiry)
	if remaining <= 0 {
		tokenStatus = "EXPIRED"
	} else {
		tokenStatus = fmt.Sprintf("valid (%s remaining)", formatDuration(remaining))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"project":        projName,
		"uid":             user.UID,
		"email":           user.Email,
		"email_verified":  user.EmailVerified,
		"display_name":    user.DisplayName,
		"disabled":        user.Disabled,
		"custom_claims":   user.CustomClaims,
		"created_at":      user.CreatedAt.Format(time.RFC3339),
		"last_sign_in":    user.LastSignIn.Format(time.RFC3339),
		"providers":       user.Providers,
		"token_status":    tokenStatus,
	})
}