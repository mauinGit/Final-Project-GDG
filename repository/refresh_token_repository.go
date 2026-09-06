package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRefreshTokenNotFound = errors.New("refresh token tidak ditemukan")

type RefreshToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	query := `INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
	          VALUES ($1, $2, $3)`
	if _, err := r.db.Exec(ctx, query, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("gagal menyimpan refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	query := `SELECT id, user_id, token_hash, expires_at, used_at, revoked_at, created_at
	          FROM refresh_tokens WHERE token_hash = $1`

	var t RefreshToken
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt,
		&t.UsedAt, &t.RevokedAt, &t.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRefreshTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil refresh token: %w", err)
	}
	return &t, nil
}

// MarkUsed menandai token sudah ditukar.
func (r *RefreshTokenRepository) MarkUsed(ctx context.Context, id int64) error {
	query := `UPDATE refresh_tokens SET used_at = now() WHERE id = $1`
	if _, err := r.db.Exec(ctx, query, id); err != nil {
		return fmt.Errorf("gagal menandai token terpakai: %w", err)
	}
	return nil
}

// Revoke mencabut satu token (dipakai saat logout).
func (r *RefreshTokenRepository) Revoke(ctx context.Context, id int64) error {
	query := `UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`
	if _, err := r.db.Exec(ctx, query, id); err != nil {
		return fmt.Errorf("gagal mencabut token: %w", err)
	}
	return nil
}

// RevokeAllForUser mencabut seluruh sesi milik satu user.
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID int64) error {
	query := `UPDATE refresh_tokens SET revoked_at = now()
	          WHERE user_id = $1 AND revoked_at IS NULL`
	if _, err := r.db.Exec(ctx, query, userID); err != nil {
		return fmt.Errorf("gagal mencabut sesi user: %w", err)
	}
	return nil
}

// DeleteExpired membersihkan token yang sudah kedaluwarsa.
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM refresh_tokens WHERE expires_at < now()`
	tag, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("gagal membersihkan token kedaluwarsa: %w", err)
	}
	return tag.RowsAffected(), nil
}