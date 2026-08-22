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

// ErrAlreadyIdentified dikembalikan saat environment sudah pernah diisi
// identitas praktikan sebelumnya.
var ErrAlreadyIdentified = errors.New("environment sudah pernah diisi identitas")

type EnvironmentRepository struct {
	db *pgxpool.Pool
}

func NewEnvironmentRepository(db *pgxpool.Pool) *EnvironmentRepository {
	return &EnvironmentRepository{db: db}
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

// IdentifyEnvironment menyimpan/update data praktikan (upsert berdasarkan NPM,
// karena praktikan bersifat persisten lintas sesi) lalu mengaitkannya ke
// environment tertentu. Mengembalikan ErrAlreadyIdentified kalau environment
// itu sudah pernah dikaitkan ke praktikan lain sebelumnya.
func (r *EnvironmentRepository) IdentifyEnvironment(ctx context.Context, environmentID, nama, npm string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // no-op kalau sudah di-Commit

	// Upsert praktikan: kalau NPM sudah ada, update nama (jaga-jaga ada typo
	// yang dibetulkan), kalau belum ada, insert baru.
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

	// Kaitkan ke environment HANYA kalau belum pernah diisi sebelumnya
	// (praktikan_id IS NULL) — mencegah environment yang sudah ada pemiliknya
	// diklaim ulang oleh orang lain.
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
		// Bukan error fatal — environment ini memang sudah diisi sebelumnya.
		return ErrAlreadyIdentified
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}