package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type readinessStub struct{ err error }

func (s readinessStub) Ready(context.Context) error { return s.err }

func TestHealthRoutes(t *testing.T) {
	for _, tc := range []struct {
		name       string
		err        error
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "live despite database failure", err: errors.New("private database error"), path: "/healthz", wantStatus: http.StatusOK, wantBody: `"status":"ok"`},
		{name: "ready", path: "/readyz", wantStatus: http.StatusOK, wantBody: `"status":"ok"`},
		{name: "database unavailable", err: errors.New("private database error"), path: "/readyz", wantStatus: http.StatusServiceUnavailable, wantBody: `"code":"DB_UNAVAILABLE"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			HealthRoutes(mux, readinessStub{err: tc.err})
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if response.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tc.wantStatus)
			}
			if !strings.Contains(response.Body.String(), tc.wantBody) {
				t.Fatalf("body = %q, want %q", response.Body.String(), tc.wantBody)
			}
			if strings.Contains(response.Body.String(), "private database error") {
				t.Fatal("response leaked internal error")
			}
			if got := response.Header().Get("Cache-Control"); got != "no-store" {
				t.Fatalf("Cache-Control = %q, want no-store", got)
			}
		})
	}
}
