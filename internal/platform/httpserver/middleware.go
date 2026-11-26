package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Mousynano/rbac-backend/internal/auth"
)

type contextKey uint8

const requestIDKey contextKey = iota

// RequestID returns the request ID assigned by the middleware.
func RequestID(r *http.Request) string {
	id, _ := r.Context().Value(requestIDKey).(string)
	return id
}

// Middleware adds bounded bodies, request IDs, safe access logs, panic recovery,
// and same-origin CSRF checks to the handler.
func Middleware(next http.Handler, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !validRequestID(id) {
			var b [16]byte
			if _, err := rand.Read(b[:]); err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			id = hex.EncodeToString(b[:])
		}
		w.Header().Set("X-Request-ID", id)
		r = r.WithContext(context.WithValue(r.Context(), requestIDKey, id))
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		started := time.Now()
		rw := &statusWriter{ResponseWriter: w}
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("http panic recovered", "request_id", id)
				if rw.status == 0 {
					http.Error(rw, "internal server error", http.StatusInternalServerError)
				}
			}
			if rw.status == 0 {
				rw.status = http.StatusOK
			}
			slog.Info("http request", "request_id", id, "method", r.Method, "path", r.URL.Path, "status", rw.status, "duration", time.Since(started))
		}()
		if mutating(r.Method) && !sameOrigin(r, allowedOrigin) {
			writeError(rw, http.StatusForbidden, "CSRF_REJECTED", "request origin rejected")
			return
		}
		if mutating(r.Method) && r.Header.Get("X-CSRF-Protection") != "1" {
			writeError(rw, http.StatusForbidden, "CSRF_REJECTED", "CSRF header required")
			return
		}
		next.ServeHTTP(rw, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}
func (w *statusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func mutating(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}
func validRequestID(id string) bool {
	if len(id) < 1 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
func sameOrigin(r *http.Request, allowed string) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Scheme+"://"+u.Host, strings.TrimRight(allowed, "/"))
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}, "meta": map[string]string{}})
}

// Authenticator verifies an opaque session cookie and returns a trusted principal.
type Authenticator interface {
	Authenticate(context.Context, string) (string, error)
}

// RequireAuth rejects requests without an authenticated principal.
func RequireAuth(authenticator Authenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("rbac_session")
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
			return
		}
		userID, err := authenticator.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			if errors.Is(err, auth.ErrUnavailable) {
				writeError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "authentication unavailable")
				return
			}
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey, userID)))
	})
}

const principalKey contextKey = 2

// PrincipalID reads the trusted user ID installed by RequireAuth.
func PrincipalID(r *http.Request) string {
	id, _ := r.Context().Value(principalKey).(string)
	return id
}
