//go:build integration

package repository

import (
	"context"
	"testing"
	"time"
)

func TestRefreshTokenRepository_CreateDanFindByHash(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRefreshTokenRepository(pool)
	ctx := context.Background()

	userID := seedUser(t, pool, "kasir@orderflow.test", "kasir")
	expires := time.Now().Add(24 * time.Hour)

	if err := repo.Create(ctx, userID, "hash-abc", expires); err != nil {
		t.Fatalf("gagal menyimpan token: %v", err)
	}

	found, err := repo.FindByHash(ctx, "hash-abc")
	if err != nil {
		t.Fatalf("gagal mengambil token: %v", err)
	}
	if found.UserID != userID {
		t.Errorf("user id salah: mau %d, dapat %d", userID, found.UserID)
	}
	if found.UsedAt != nil {
		t.Error("token baru harusnya belum terpakai")
	}
	if found.RevokedAt != nil {
		t.Error("token baru harusnya belum dicabut")
	}
}

func TestRefreshTokenRepository_HashTidakDikenal(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRefreshTokenRepository(pool)

	_, err := repo.FindByHash(context.Background(), "hash-yang-tidak-ada")
	if err != ErrRefreshTokenNotFound {
		t.Errorf("harusnya ErrRefreshTokenNotFound, dapat: %v", err)
	}
}

func TestRefreshTokenRepository_MarkUsed(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRefreshTokenRepository(pool)
	ctx := context.Background()

	userID := seedUser(t, pool, "kasir@orderflow.test", "kasir")
	repo.Create(ctx, userID, "hash-abc", time.Now().Add(time.Hour))

	stored, _ := repo.FindByHash(ctx, "hash-abc")
	if err := repo.MarkUsed(ctx, stored.ID); err != nil {
		t.Fatalf("gagal menandai terpakai: %v", err)
	}

	after, _ := repo.FindByHash(ctx, "hash-abc")
	if after.UsedAt == nil {
		t.Error("used_at harusnya terisi setelah MarkUsed")
	}
}

// Inti reuse detection: satu pelanggaran mencabut seluruh sesi user.
func TestRefreshTokenRepository_RevokeAllForUser(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRefreshTokenRepository(pool)
	ctx := context.Background()

	korban := seedUser(t, pool, "korban@orderflow.test", "kasir")
	lain := seedUser(t, pool, "lain@orderflow.test", "koki")

	repo.Create(ctx, korban, "hash-1", time.Now().Add(time.Hour))
	repo.Create(ctx, korban, "hash-2", time.Now().Add(time.Hour))
	repo.Create(ctx, korban, "hash-3", time.Now().Add(time.Hour))
	repo.Create(ctx, lain, "hash-lain", time.Now().Add(time.Hour))

	if err := repo.RevokeAllForUser(ctx, korban); err != nil {
		t.Fatalf("gagal mencabut sesi: %v", err)
	}

	for _, h := range []string{"hash-1", "hash-2", "hash-3"} {
		token, err := repo.FindByHash(ctx, h)
		if err != nil {
			t.Fatalf("gagal mengambil %s: %v", h, err)
		}
		if token.RevokedAt == nil {
			t.Errorf("%s harusnya ikut dicabut", h)
		}
	}

	// Sesi user lain tidak boleh terpengaruh.
	tokenLain, err := repo.FindByHash(ctx, "hash-lain")
	if err != nil {
		t.Fatalf("gagal mengambil token user lain: %v", err)
	}
	if tokenLain.RevokedAt != nil {
		t.Error("sesi user lain tidak boleh ikut dicabut")
	}
}

func TestRefreshTokenRepository_RevokeTidakMenimpaWaktuPertama(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRefreshTokenRepository(pool)
	ctx := context.Background()

	userID := seedUser(t, pool, "kasir@orderflow.test", "kasir")
	repo.Create(ctx, userID, "hash-abc", time.Now().Add(time.Hour))

	stored, _ := repo.FindByHash(ctx, "hash-abc")
	repo.Revoke(ctx, stored.ID)

	pertama, _ := repo.FindByHash(ctx, "hash-abc")
	waktuPertama := *pertama.RevokedAt

	time.Sleep(50 * time.Millisecond)
	repo.Revoke(ctx, stored.ID)

	kedua, _ := repo.FindByHash(ctx, "hash-abc")
	if !kedua.RevokedAt.Equal(waktuPertama) {
		t.Error("pencabutan kedua tidak boleh menimpa waktu pencabutan pertama")
	}
}

func TestRefreshTokenRepository_DeleteExpired(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRefreshTokenRepository(pool)
	ctx := context.Background()

	userID := seedUser(t, pool, "kasir@orderflow.test", "kasir")

	repo.Create(ctx, userID, "hash-aktif", time.Now().Add(time.Hour))
	repo.Create(ctx, userID, "hash-basi-1", time.Now().Add(-time.Hour))
	repo.Create(ctx, userID, "hash-basi-2", time.Now().Add(-48*time.Hour))

	n, err := repo.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("gagal membersihkan: %v", err)
	}
	if n != 2 {
		t.Errorf("harusnya 2 token dihapus, dapat %d", n)
	}

	if _, err := repo.FindByHash(ctx, "hash-aktif"); err != nil {
		t.Errorf("token aktif tidak boleh ikut terhapus: %v", err)
	}
	if _, err := repo.FindByHash(ctx, "hash-basi-1"); err != ErrRefreshTokenNotFound {
		t.Error("token kedaluwarsa harusnya sudah terhapus")
	}
}