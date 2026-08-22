package handler

import (
	"log/slog"
	"net/http"
	"time"

	"lxd-dev/internal/middleware"
	"lxd-dev/internal/repository"
)

// NewRouter merakit seluruh route aplikasi. Sengaja pakai net/http.ServeMux
// bawaan Go 1.22 (sudah mendukung pola "METHOD /path" dan path parameter)
// daripada router pihak ketiga (chi, gorilla/mux, dll) — untuk kebutuhan
// route sesederhana ini, dependency tambahan cuma nambah overhead tanpa
// manfaat berarti.
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

// withLogging membungkus seluruh handler dengan structured logging ringan
// (log/slog bawaan Go, tanpa dependency logging pihak ketiga) supaya tiap
// request tercatat: method, path, durasi, dan status.
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
// bisa ditangkap untuk keperluan logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}