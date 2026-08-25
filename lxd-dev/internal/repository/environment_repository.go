package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lxd-dev/internal/models"
)

// ErrNotFound dikembalikan saat token tidak cocok dengan environment manapun.
var ErrNotFound = errors.New("environment tidak ditemukan")

// ErrIdentityMismatch dikembalikan saat environment sudah pernah diisi oleh
// praktikan lain, dan NPM yang di-submit sekarang TIDAK cocok dengan yang
// tercatat sebelumnya. Ini mencegah orang lain (misal teman sekelas yang
// numpang PC) mengklaim akses ke environment yang bukan miliknya.
var ErrIdentityMismatch = errors.New("nama/npm tidak cocok dengan environment ini")

type EnvironmentRepository struct {
	db *pgxpool.Pool
}

func NewEnvironmentRepository(db *pgxpool.Pool) *EnvironmentRepository {
	return &EnvironmentRepository{db: db}
}

// IdentifyEnvironment menangani 2 skenario:
//  1. Environment BELUM pernah diisi (praktikan_id masih NULL): upsert
//     praktikan baru (atau yang sudah ada berdasarkan NPM), lalu kaitkan ke
//     environment ini.
//  2. Environment SUDAH pernah diisi: verifikasi NPM yang di-submit sekarang
//     cocok dengan NPM yang sudah tercatat. Kalau cocok, dianggap berhasil
//     (tanpa mengubah apapun). Kalau tidak cocok, tolak dengan ErrIdentityMismatch.
func (r *EnvironmentRepository) IdentifyEnvironment(ctx context.Context, environmentID, nama, npm string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // no-op kalau sudah di-Commit

	// Kunci baris environment ini dan cek NPM praktikan yang sudah terkait
	// (kalau ada), sebelum memutuskan mau upsert baru atau verifikasi.
	//
	// PENTING: "FOR UPDATE OF e" (bukan "FOR UPDATE" polos) — PostgreSQL
	// tidak mengizinkan FOR UPDATE menyasar ke sisi nullable dari LEFT JOIN
	// (di sini tabel praktikan), jadi harus dibatasi eksplisit hanya
	// mengunci baris di tabel environments.
	const checkQuery = `
		SELECT p.npm
		FROM environments e
		LEFT JOIN praktikan p ON p.id = e.praktikan_id
		WHERE e.id = $1
		FOR UPDATE OF e
	`
	var existingNPM *string
	if err := tx.QueryRow(ctx, checkQuery, environmentID).Scan(&existingNPM); err != nil {
		return fmt.Errorf("cek environment existing: %w", err)
	}

	if existingNPM != nil {
		// Sudah pernah diisi -> WAJIB verifikasi, bukan otomatis lolos.
		if *existingNPM != npm {
			return ErrIdentityMismatch
		}
		// NPM cocok, tidak ada yang perlu diubah.
		return tx.Commit(ctx)
	}

	// Belum pernah diisi -> upsert praktikan (berdasarkan NPM, karena
	// praktikan bersifat persisten lintas sesi) lalu kaitkan ke environment.
	var praktikanID string
	const upsertQuery = `
		INSERT INTO praktikan (npm, nama)
		VALUES ($1, $2)
		ON CONFLICT (npm) DO UPDATE SET nama = EXCLUDED.nama, updated_at = now()
		RETURNING id
	`
	if err := tx.QueryRow(ctx, upsertQuery, npm, nama).Scan(&praktikanID); err != nil {
		return fmt.Errorf("upsert praktikan: %w", err)
	}

	const linkQuery = `
		UPDATE environments
		SET praktikan_id = $1, identified_at = now()
		WHERE id = $2 AND praktikan_id IS NULL
	`
	tag, err := tx.Exec(ctx, linkQuery, praktikanID, environmentID)
	if err != nil {
		return fmt.Errorf("link praktikan ke environment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Race condition: environment ini baru saja diisi proses lain
		// tepat setelah pengecekan di atas. Aman untuk dianggap konflik.
		return ErrIdentityMismatch
	}

	return tx.Commit(ctx)
}

// GetByTokenHash mengambil detail environment (join sessions, rooms, modules)
// berdasarkan hash token yang dikirim TUI lewat header Authorization.
func (r *EnvironmentRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.EnvironmentDetail, error) {
	const query = `
		SELECT
			e.id,
			e.container_name,
			s.course_code,
			m.code AS module,
			rm.nama AS room,
			s.meeting_number,
			s.session_date,
			e.status,
			(e.praktikan_id IS NOT NULL) AS already_identified
		FROM environments e
		JOIN sessions s ON s.id = e.session_id
		JOIN rooms rm   ON rm.id = s.room_id
		JOIN modules m  ON m.id = s.module_id
		WHERE e.api_token_hash = $1
	`

	var d models.EnvironmentDetail
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&d.ID,
		&d.ContainerName,
		&d.CourseCode,
		&d.Module,
		&d.Room,
		&d.MeetingNumber,
		&d.SessionDate,
		&d.Status,
		&d.AlreadyIdentified,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query environment by token: %w", err)
	}

	return &d, nil
}