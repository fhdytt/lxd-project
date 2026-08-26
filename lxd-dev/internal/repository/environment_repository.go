package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lxd-dev/internal/models"
)

// ErrNotFound dikembalikan saat token tidak cocok dengan environment manapun.
var ErrNotFound = errors.New("environment tidak ditemukan")

/*
 ErrIdentityMismatch dikembalikan saat environment sudah pernah diisi oleh
 praktikan, mencegah orang lain mengakses ke environment yang bukan miliknya.
*/
var ErrIdentityMismatch = errors.New("nama/npm tidak cocok dengan environment ini")

type EnvironmentRepository struct {
	db *pgxpool.Pool
}

func NewEnvironmentRepository(db *pgxpool.Pool) *EnvironmentRepository {
	return &EnvironmentRepository{db: db}
}

/*
 IdentifyEnvironment menangani 2 kemungkinan :
  1. Environment BELUM pernah diisi maka praktikan baru dikaitkan ke environment ini
  2. Environment SUDAH pernah diisi: verifikasi nama DAN NPM yang di-submit
	 sekarang SAMA-SAMA cocok dengan yang sudah tercatat. Kalau salah satu saja
	 beda, ditolak. Kemudian Perbandingan nama dibuat case-insensitive dan
	 mengabaikan spasi. 
*/
func (r *EnvironmentRepository) IdentifyEnvironment(ctx context.Context, environmentID, nama, npm string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) 

	// Kunci baris environment ini dan cek nama+NPM praktikan yang sudah tercatat sebelum memutuskan mau upsert baru atau verifikasi.
	const checkQuery = `
		SELECT p.npm, p.nama
		FROM environments e
		LEFT JOIN praktikan p ON p.id = e.praktikan_id
		WHERE e.id = $1
		FOR UPDATE OF e
	`
	var existingNPM, existingNama *string
	if err := tx.QueryRow(ctx, checkQuery, environmentID).Scan(&existingNPM, &existingNama); err != nil {
		return fmt.Errorf("cek environment existing: %w", err)
	}

	if existingNPM != nil {
		// Jika sudah pernah mengisi, WAJIB verifikasi nama DAN NPM
		npmMatch := *existingNPM == npm
		namaMatch := strings.EqualFold(strings.TrimSpace(*existingNama), strings.TrimSpace(nama))

		if !npmMatch || !namaMatch {
			return ErrIdentityMismatch
		}
		return tx.Commit(ctx)
	}

	// Jika belum pernah diisi, maka akan upsert praktikan dan kaitkan ke environment.
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
		return ErrIdentityMismatch
	}

	return tx.Commit(ctx)
}

// GetByTokenHash mengambil detail environment berdasarkan hash token yang dikirim TUI lewat header Authorization
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