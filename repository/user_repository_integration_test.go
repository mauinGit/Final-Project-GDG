//go:build integration

package repository

import (
	"context"
	"testing"
)

func TestUserRepository_CreateDanFindByEmail(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	created, err := repo.Create(ctx, "kasir@orderflow.test", "hash-rahasia", "kasir")
	if err != nil {
		t.Fatalf("gagal menyimpan user: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("id harusnya terisi")
	}

	found, err := repo.FindByEmail(ctx, "kasir@orderflow.test")
	if err != nil {
		t.Fatalf("gagal mengambil user: %v", err)
	}
	if found.Role != "kasir" {
		t.Errorf("role salah: dapat %s", found.Role)
	}
	if found.PasswordHash != "hash-rahasia" {
		t.Error("password hash tidak cocok")
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "koki@orderflow.test", "hash", "koki")

	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("gagal mengambil user: %v", err)
	}
	if found.Email != "koki@orderflow.test" {
		t.Errorf("email salah: dapat %s", found.Email)
	}
}

func TestUserRepository_TidakDitemukan(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	if _, err := repo.FindByEmail(ctx, "hantu@orderflow.test"); err != ErrUserNotFound {
		t.Errorf("FindByEmail harusnya ErrUserNotFound, dapat: %v", err)
	}
	if _, err := repo.FindByID(ctx, 9999); err != ErrUserNotFound {
		t.Errorf("FindByID harusnya ErrUserNotFound, dapat: %v", err)
	}
}

func TestUserRepository_EmailUnik(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	if _, err := repo.Create(ctx, "kasir@orderflow.test", "hash", "kasir"); err != nil {
		t.Fatalf("insert pertama harusnya berhasil: %v", err)
	}

	_, err := repo.Create(ctx, "kasir@orderflow.test", "hash-lain", "koki")
	if err == nil {
		t.Fatal("email duplikat harusnya ditolak")
	}
	if !isUniqueViolationErr(err) {
		t.Errorf("harusnya pelanggaran unique, dapat: %v", err)
	}
}

// Constraint chk_users_role hanya mengizinkan 'kasir' dan 'koki'.
func TestUserRepository_RoleDibatasiConstraint(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)

	_, err := repo.Create(context.Background(), "admin@orderflow.test", "hash", "admin")
	if err == nil {
		t.Fatal("role di luar kasir/koki harusnya ditolak database")
	}
}

func TestUserRepository_ExistsByEmail(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	repo.Create(ctx, "kasir@orderflow.test", "hash", "kasir")

	ada, err := repo.ExistsByEmail(ctx, "kasir@orderflow.test")
	if err != nil {
		t.Fatalf("gagal mengecek: %v", err)
	}
	if !ada {
		t.Error("email yang sudah ada harusnya true")
	}

	tidakAda, err := repo.ExistsByEmail(ctx, "hantu@orderflow.test")
	if err != nil {
		t.Fatalf("gagal mengecek: %v", err)
	}
	if tidakAda {
		t.Error("email yang tidak ada harusnya false")
	}
}