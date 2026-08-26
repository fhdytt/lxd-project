package handler

import (
	"log/slog"
	"net/http"
	"time"

	"lxd-dev/internal/middleware"
	"lxd-dev/internal/repository"
)

func NewRouter(envRepo *repository.EnvironmentRepository) http.Handler {
	mux := http.NewServeMux()

	envHandler := NewEnvironmentHandler(envRepo)
	authMiddleware := middleware.Auth(envRepo)

	mux.Handle("GET /api/v1/environments/me", authMiddleware(http.HandlerFunc(envHandler.GetMe)))
	mux.Handle("POST /api/v1/environments/me/identify", authMiddleware(http.HandlerFunc(envHandler.Identify)))

	mux.HandleFunc("GET /healthz", healthCheck)

	return withLogging(mux)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// withLogging membungkus seluruh handler
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// statusRecorder membungkus http.ResponseWriter supaya status code response
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}