package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Readiness is the application capability used by the HTTP health adapter.
type Readiness interface {
	Ready(context.Context) error
}

type healthResponse struct {
	Data  *healthData  `json:"data,omitempty"`
	Error *healthError `json:"error,omitempty"`
	Meta  healthMeta   `json:"meta"`
}

type healthData struct {
	Status string `json:"status"`
}
type healthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type healthMeta struct {
	Time string `json:"time"`
}

// HealthRoutes registers public liveness and readiness routes.
func HealthRoutes(mux *http.ServeMux, readiness Readiness) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeHealth(w, http.StatusOK, healthResponse{Data: &healthData{Status: "ok"}})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := readiness.Ready(ctx); err != nil {
			slog.Warn("readiness check failed", "err", err)
			writeHealth(w, http.StatusServiceUnavailable, healthResponse{
				Error: &healthError{Code: "DB_UNAVAILABLE", Message: "database not ready"},
			})
			return
		}
		writeHealth(w, http.StatusOK, healthResponse{Data: &healthData{Status: "ok"}})
	})
}

func writeHealth(w http.ResponseWriter, status int, response healthResponse) {
	response.Meta.Time = time.Now().UTC().Format(time.RFC3339)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("write health response", "err", err)
	}
}
