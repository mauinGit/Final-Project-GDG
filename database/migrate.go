package database

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, path string) error {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("gagal membaca file migrasi: %w", err)
	}

	if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("gagal menjalankan migrasi: %w", err)
	}

	return nil
}