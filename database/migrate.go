package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations menjalankan semua file .sql di dalam folder secara berurutan.
// File yang sudah pernah dijalankan akan dilewati.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	// Tabel pencatat migrasi yang sudah dijalankan.
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("gagal menyiapkan tabel migrasi: %w", err)
	}

	// Kumpulkan versi yang sudah pernah dijalankan.
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("gagal membaca riwayat migrasi: %w", err)
	}

	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return fmt.Errorf("gagal membaca versi migrasi: %w", err)
		}
		applied[v] = true
	}
	rows.Close()

	// Baca semua file .sql lalu urutkan berdasarkan nama.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("gagal membaca folder migrasi: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		if applied[name] {
			continue
		}

		sqlBytes, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("gagal membaca %s: %w", name, err)
		}

		// Satu file = satu transaksi. Kalau gagal di tengah,
		// tidak ada perubahan yang tertinggal setengah jadi.
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("gagal memulai transaksi migrasi: %w", err)
		}

		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("gagal menjalankan migrasi %s: %w", name, err)
		}

		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("gagal mencatat migrasi %s: %w", name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("gagal commit migrasi %s: %w", name, err)
		}
	}

	return nil
}