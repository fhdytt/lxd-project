package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"lxd-dev/internal/middleware"
	"lxd-dev/internal/models"
	"lxd-dev/internal/repository"
)

type EnvironmentHandler struct {
	repo *repository.EnvironmentRepository
}

func NewEnvironmentHandler(repo *repository.EnvironmentRepository) *EnvironmentHandler {
	return &EnvironmentHandler{repo: repo}
}

// GetMe menangani GET /api/v1/environments/me
// Environment yang relevan sudah diambil oleh middleware Auth, handler ini
// tinggal serialize ke JSON.
func (h *EnvironmentHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	env, ok := middleware.EnvironmentFromContext(r.Context())
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, env)
}

// Identify menangani POST /api/v1/environments/me/identify
func (h *EnvironmentHandler) Identify(w http.ResponseWriter, r *http.Request) {
	env, ok := middleware.EnvironmentFromContext(r.Context())
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var req models.IdentifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "body request tidak valid", http.StatusBadRequest)
		return
	}

	if req.Nama == "" || req.NPM == "" {
		http.Error(w, "nama dan npm wajib diisi", http.StatusBadRequest)
		return
	}

	err := h.repo.IdentifyEnvironment(r.Context(), env.ID, req.Nama, req.NPM)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})

	case errors.Is(err, repository.ErrIdentityMismatch):
		// Environment ini sudah terdaftar milik praktikan lain, dan
		// nama/NPM yang di-submit sekarang tidak cocok. Ini KEAMANAN,
		// bukan sekadar info — mencegah orang lain mengklaim akses ke
		// environment yang bukan miliknya.
		http.Error(w, "nama/NPM tidak cocok dengan environment ini", http.StatusForbidden)

	default:
		slog.Error("gagal menyimpan identifikasi praktikan", "error", err, "environment_id", env.ID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("gagal menulis response JSON", "error", err)
	}
}