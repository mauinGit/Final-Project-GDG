package service

import (
	"context"
	"errors"
	"testing"
	"time"

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

func (m *mockUserRepo) FindByID(ctx context.Context, id int64) (*models.User, error) {
	return m.user, m.err
}

type mockRefreshRepo struct {
	stored    *repository.RefreshToken
	findErr   error
	createErr error

	markedUsedID  int64
	revokedID     int64
	revokedAllFor int64
}

func (m *mockRefreshRepo) Create(ctx context.Context, userID int64, hash string, exp time.Time) error {
	return m.createErr
}

func (m *mockRefreshRepo) FindByHash(ctx context.Context, hash string) (*repository.RefreshToken, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.stored, nil
}

func (m *mockRefreshRepo) MarkUsed(ctx context.Context, id int64) error {
	m.markedUsedID = id
	return nil
}

func (m *mockRefreshRepo) Revoke(ctx context.Context, id int64) error {
	m.revokedID = id
	return nil
}

func (m *mockRefreshRepo) RevokeAllForUser(ctx context.Context, userID int64) error {
	m.revokedAllFor = userID
	return nil
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

// Login

func TestLogin_Berhasil(t *testing.T) {
	user := buatUserAsli(t, "password123")
	svc := NewAuthService(&mockUserRepo{user: user}, &mockRefreshRepo{}, "rahasia-test")

	result, err := svc.Login(context.Background(), "kasir@orderflow.com", "password123")

	if err != nil {
		t.Fatalf("harusnya berhasil, tapi dapat error: %v", err)
	}
	if result.Token == "" {
		t.Error("token tidak boleh kosong")
	}
	if result.RefreshToken == "" {
		t.Error("refresh token tidak boleh kosong")
	}
	if result.Role != "kasir" {
		t.Errorf("role salah: mau 'kasir', dapat '%s'", result.Role)
	}
}

func TestLogin_PasswordSalah(t *testing.T) {
	user := buatUserAsli(t, "password123")
	svc := NewAuthService(&mockUserRepo{user: user}, &mockRefreshRepo{}, "rahasia-test")

	_, err := svc.Login(context.Background(), "kasir@orderflow.com", "password-salah")

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("harusnya ErrInvalidCredentials, dapat: %v", err)
	}
}

func TestLogin_UserTidakDitemukan(t *testing.T) {
	repo := &mockUserRepo{err: repository.ErrUserNotFound}
	svc := NewAuthService(repo, &mockRefreshRepo{}, "rahasia-test")

	_, err := svc.Login(context.Background(), "tidakada@orderflow.com", "password123")

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("harusnya ErrInvalidCredentials, dapat: %v", err)
	}
}

func TestLogin_ErrorDatabase(t *testing.T) {
	repo := &mockUserRepo{err: errors.New("koneksi database putus")}
	svc := NewAuthService(repo, &mockRefreshRepo{}, "rahasia-test")

	_, err := svc.Login(context.Background(), "kasir@orderflow.com", "password123")

	if err == nil {
		t.Error("harusnya error database diteruskan, bukan nil")
	}
	if errors.Is(err, ErrInvalidCredentials) {
		t.Error("error database jangan disamarkan jadi ErrInvalidCredentials")
	}
}

// Refresh

func TestRefresh_Berhasil(t *testing.T) {
	user := buatUserAsli(t, "password123")
	refreshRepo := &mockRefreshRepo{
		stored: &repository.RefreshToken{
			ID:        7,
			UserID:    1,
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}
	svc := NewAuthService(&mockUserRepo{user: user}, refreshRepo, "rahasia-test")

	result, err := svc.Refresh(context.Background(), "token-apa-saja")
	if err != nil {
		t.Fatalf("harusnya berhasil, dapat: %v", err)
	}
	if result.Token == "" || result.RefreshToken == "" {
		t.Error("harusnya menerbitkan pasangan token baru")
	}
	if refreshRepo.markedUsedID != 7 {
		t.Errorf("token lama harus ditandai terpakai, dapat id %d", refreshRepo.markedUsedID)
	}
}

func TestRefresh_TokenTidakDikenal(t *testing.T) {
	refreshRepo := &mockRefreshRepo{findErr: repository.ErrRefreshTokenNotFound}
	svc := NewAuthService(&mockUserRepo{}, refreshRepo, "rahasia-test")

	_, err := svc.Refresh(context.Background(), "token-palsu")
	if !errors.Is(err, ErrInvalidRefresh) {
		t.Errorf("harusnya ErrInvalidRefresh, dapat: %v", err)
	}
}

func TestRefresh_TokenSudahDipakai_CabutSemuaSesi(t *testing.T) {
	used := time.Now().Add(-time.Minute)
	refreshRepo := &mockRefreshRepo{
		stored: &repository.RefreshToken{
			ID:        7,
			UserID:    42,
			ExpiresAt: time.Now().Add(time.Hour),
			UsedAt:    &used,
		},
	}
	svc := NewAuthService(&mockUserRepo{}, refreshRepo, "rahasia-test")

	_, err := svc.Refresh(context.Background(), "token-bocor")
	if !errors.Is(err, ErrRefreshReused) {
		t.Errorf("harusnya ErrRefreshReused, dapat: %v", err)
	}
	if refreshRepo.revokedAllFor != 42 {
		t.Errorf("seluruh sesi user harus dicabut, dapat userID %d", refreshRepo.revokedAllFor)
	}
}

func TestRefresh_TokenSudahDicabut_CabutSemuaSesi(t *testing.T) {
	revoked := time.Now().Add(-time.Minute)
	refreshRepo := &mockRefreshRepo{
		stored: &repository.RefreshToken{
			ID:        8,
			UserID:    42,
			ExpiresAt: time.Now().Add(time.Hour),
			RevokedAt: &revoked,
		},
	}
	svc := NewAuthService(&mockUserRepo{}, refreshRepo, "rahasia-test")

	_, err := svc.Refresh(context.Background(), "token-dicabut")
	if !errors.Is(err, ErrRefreshReused) {
		t.Errorf("harusnya ErrRefreshReused, dapat: %v", err)
	}
	if refreshRepo.revokedAllFor != 42 {
		t.Errorf("seluruh sesi user harus dicabut, dapat userID %d", refreshRepo.revokedAllFor)
	}
}

func TestRefresh_TokenKedaluwarsa(t *testing.T) {
	refreshRepo := &mockRefreshRepo{
		stored: &repository.RefreshToken{
			ID:        9,
			UserID:    1,
			ExpiresAt: time.Now().Add(-time.Hour),
		},
	}
	svc := NewAuthService(&mockUserRepo{}, refreshRepo, "rahasia-test")

	_, err := svc.Refresh(context.Background(), "token-lama")
	if !errors.Is(err, ErrRefreshExpired) {
		t.Errorf("harusnya ErrRefreshExpired, dapat: %v", err)
	}
}

// Logout

func TestLogout_MencabutToken(t *testing.T) {
	refreshRepo := &mockRefreshRepo{
		stored: &repository.RefreshToken{ID: 11, UserID: 1},
	}
	svc := NewAuthService(&mockUserRepo{}, refreshRepo, "rahasia-test")

	if err := svc.Logout(context.Background(), "token"); err != nil {
		t.Fatalf("harusnya berhasil, dapat: %v", err)
	}
	if refreshRepo.revokedID != 11 {
		t.Errorf("token harus dicabut, dapat id %d", refreshRepo.revokedID)
	}
}

func TestLogout_TokenTidakDikenal(t *testing.T) {
	refreshRepo := &mockRefreshRepo{findErr: repository.ErrRefreshTokenNotFound}
	svc := NewAuthService(&mockUserRepo{}, refreshRepo, "rahasia-test")

	err := svc.Logout(context.Background(), "token-palsu")
	if !errors.Is(err, ErrInvalidRefresh) {
		t.Errorf("harusnya ErrInvalidRefresh, dapat: %v", err)
	}
}