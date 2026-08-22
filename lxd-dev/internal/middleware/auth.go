package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"lxd-dev/internal/models"
	"lxd-dev/internal/repository"
)

type contextKey string

const environmentContextKey contextKey = "environment"

// HashToken mengubah token plaintext (yang dikirim TUI) menjadi hash SHA-256
// heksadesimal, format yang sama seperti yang disimpan di kolom
// environments.api_token_hash. SHA-256 dipilih (bukan bcrypt) karena token
// ini bukan password manusia — kita butuh lookup cepat dan deterministik
// langsung lewat query database, bukan hash lambat anti-brute-force.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Auth adalah middleware yang memvalidasi header "Authorization: Bearer <token>",
// mencari environment yang cocok, lalu menyisipkan detail environment tersebut
// ke context request supaya handler berikutnya tidak perlu query ulang.
func Auth(repo *repository.EnvironmentRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := extractBearerToken(r.Header.Get("Authorization"))
			if err != nil {
				writeUnauthorized(w, "header Authorization tidak valid")
				return
			}

			env, err := repo.GetByTokenHash(r.Context(), HashToken(token))
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					writeUnauthorized(w, "token tidak dikenali")
					return
				}
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), environmentContextKey, env)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// EnvironmentFromContext mengambil environment yang sudah divalidasi middleware
// Auth. Handler HARUS berada di belakang middleware Auth, kalau tidak akan
// mengembalikan ok=false.
func EnvironmentFromContext(ctx context.Context) (*models.EnvironmentDetail, bool) {
	env, ok := ctx.Value(environmentContextKey).(*models.EnvironmentDetail)
	return env, ok
}

func extractBearerToken(headerValue string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(headerValue, prefix) {
		return "", errors.New("missing bearer prefix")
	}
	token := strings.TrimSpace(strings.TrimPrefix(headerValue, prefix))
	if token == "" {
		return "", errors.New("empty token")
	}
	return token, nil
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	http.Error(w, message, http.StatusUnauthorized)
}