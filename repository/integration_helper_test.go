//go:build integration

package repository

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB menyalakan PostgreSQL sementara, menjalankan semua migrasi,
// lalu mengembalikan pool yang siap dipakai.
// Container otomatis dihapus setelah test selesai.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("orderflow_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("gagal menyalakan container postgres: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("gagal menghentikan container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("gagal mengambil connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("gagal membuat pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("gagal ping database: %v", err)
	}

	runMigrationsForTest(t, pool)
	return pool
}

// runMigrationsForTest menjalankan semua file .sql di folder migrations.
func runMigrationsForTest(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	dir := filepath.Join("..", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("gagal membaca folder migrasi: %v", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	ctx := context.Background()
	for _, name := range files {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("gagal membaca %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			t.Fatalf("gagal menjalankan migrasi %s: %v", name, err)
		}
	}
}

// seedUser menyisipkan satu user untuk keperluan test.
func seedUser(t *testing.T, pool *pgxpool.Pool, email, role string) int64 {
	t.Helper()

	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (email, password_hash, role) VALUES ($1, 'hash', $2) RETURNING id`,
		email, role).Scan(&id)
	if err != nil {
		t.Fatalf("gagal seed user: %v", err)
	}
	return id
}

func isUniqueViolationErr(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolationErr(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}