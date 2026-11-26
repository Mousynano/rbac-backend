package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/Mousynano/rbac-backend/internal/auth"
)

// CurrentUserReader returns the public profile for a trusted user ID.
type CurrentUserReader interface {
	CurrentUser(context.Context, string) (auth.UserProfile, error)
}

// CurrentUserRoute registers GET /api/v1/auth/me behind the session guard.
func CurrentUserRoute(mux *http.ServeMux, auth Authenticator, users CurrentUserReader) {
	mux.Handle("GET /api/v1/auth/me", RequireAuth(auth, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		user, err := users.CurrentUser(ctx, PrincipalID(r))
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "USER_UNAVAILABLE", "user profile unavailable")
			return
		}
		WriteJSON(w, http.StatusOK, user)
	})))
}
