package service

import (
	"context"
	"errors"
	"testing"

	"FinalProjectBE/models"
	"FinalProjectBE/repository"
	"FinalProjectBE/utils"
)

type mockUserRepo struct {
	user *models.User
	err  error
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return m.user, m.err
}

// buatUserAsli membantu menyiapkan user dengan password yang sudah di-hash.
func buatUserAsli(t *testing.T, password string) *models.User {
	hash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("gagal hash password untuk setup test: %v", err)
	}
	return &models.User{
		ID:           1,
		Email:        "kasir@orderflow.com",
		PasswordHash: hash,
		Role:         "kasir",
	}
}

// login dengan kredensial benar → dapat token & role.
func TestLogin_Berhasil(t *testing.T) {
	user := buatUserAsli(t, "password123")
	repo := &mockUserRepo{user: user, err: nil}
	svc := NewAuthService(repo, "rahasia-test")

	result, err := svc.Login(context.Background(), "kasir@orderflow.com", "password123")

	if err != nil {
		t.Fatalf("harusnya berhasil, tapi dapat error: %v", err)
	}
	if result.Token == "" {
		t.Error("token tidak boleh kosong")
	}
	if result.Role != "kasir" {
		t.Errorf("role salah: mau 'kasir', dapat '%s'", result.Role)
	}
}

// password salah → ditolak dengan ErrInvalidCredentials.
func TestLogin_PasswordSalah(t *testing.T) {
	user := buatUserAsli(t, "password123")
	repo := &mockUserRepo{user: user, err: nil}
	svc := NewAuthService(repo, "rahasia-test")

	_, err := svc.Login(context.Background(), "kasir@orderflow.com", "password-salah")

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("harusnya ErrInvalidCredentials, dapat: %v", err)
	}
}

// email tidak terdaftar → ditolak dengan ErrInvalidCredentials.
func TestLogin_UserTidakDitemukan(t *testing.T) {
	repo := &mockUserRepo{user: nil, err: repository.ErrUserNotFound}
	svc := NewAuthService(repo, "rahasia-test")

	_, err := svc.Login(context.Background(), "tidakada@orderflow.com", "password123")

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("harusnya ErrInvalidCredentials, dapat: %v", err)
	}
}

// error database selain user-not-found → diteruskan apa adanya.
func TestLogin_ErrorDatabase(t *testing.T) {
	repo := &mockUserRepo{user: nil, err: errors.New("koneksi database putus")}
	svc := NewAuthService(repo, "rahasia-test")

	_, err := svc.Login(context.Background(), "kasir@orderflow.com", "password123")

	if err == nil {
		t.Error("harusnya error database diteruskan, bukan nil")
	}
	if errors.Is(err, ErrInvalidCredentials) {
		t.Error("error database jangan disamarkan jadi ErrInvalidCredentials")
	}
}