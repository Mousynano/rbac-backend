// Package httpapi adapts authentication use cases to HTTP.
package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/Mousynano/rbac-backend/internal/auth"
	"github.com/Mousynano/rbac-backend/internal/platform/httpserver"
)

func Routes(mux *http.ServeMux, service auth.Service, secure bool) {
	mux.HandleFunc("POST /api/v1/auth/register", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := httpserver.DecodeJSON(r, &in); err != nil {
			writeError(w, 400, "INVALID_REQUEST")
			return
		}
		pending, err := service.StartPasswordFlow(r.Context(), in.Email, in.Password, "register")
		if err != nil {
			// Registration is intentionally generic to avoid account enumeration.
			var raw [32]byte
			if _, randomErr := rand.Read(raw[:]); randomErr != nil {
				writeError(w, 503, "AUTH_UNAVAILABLE")
				return
			}
			setPendingCookie(w, base64.RawURLEncoding.EncodeToString(raw[:]), secure)
			httpserver.WriteJSON(w, 202, map[string]string{"status": "otp_required"})
			return
		}
		setPendingCookie(w, pending.Token, secure)
		httpserver.WriteJSON(w, 202, map[string]string{"status": "otp_required"})
	})
	mux.HandleFunc("POST /api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := httpserver.DecodeJSON(r, &in); err != nil {
			writeError(w, 400, "INVALID_REQUEST")
			return
		}
		pending, err := service.StartPasswordFlow(r.Context(), in.Email, in.Password, "login")
		if err != nil {
			writeError(w, 401, "INVALID_CREDENTIALS")
			return
		}
		setPendingCookie(w, pending.Token, secure)
		httpserver.WriteJSON(w, 202, map[string]string{"status": "otp_required"})
	})
	mux.HandleFunc("POST /api/v1/auth/otp/verify", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("rbac_pending")
		if err != nil {
			writeError(w, 401, "CHALLENGE_INVALID")
			return
		}
		var in struct {
			Code string `json:"code"`
		}
		if err = httpserver.DecodeJSON(r, &in); err != nil {
			writeError(w, 400, "INVALID_REQUEST")
			return
		}
		user, token, err := service.Verify(r.Context(), cookie.Value, in.Code)
		if err != nil {
			if errors.Is(err, auth.ErrCredentials) {
				writeError(w, 401, "CHALLENGE_INVALID")
			} else {
				writeError(w, 503, "AUTH_UNAVAILABLE")
			}
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "rbac_session", Value: token, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(24 * time.Hour), MaxAge: 86400})
		http.SetCookie(w, &http.Cookie{Name: "rbac_pending", Path: "/api/v1/auth", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
		httpserver.WriteJSON(w, 200, user)
	})
}
func setPendingCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: "rbac_pending", Value: token, Path: "/api/v1/auth", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: 900})
}
func writeError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":{"code":"` + code + `","message":"request could not be completed"},"meta":{}}`))
}
