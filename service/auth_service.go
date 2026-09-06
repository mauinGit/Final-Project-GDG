package service

import (
	"context"
	"errors"
	"time"

	"FinalProjectBE/models"
	"FinalProjectBE/repository"
	"FinalProjectBE/utils"
)

var (
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrInvalidRefresh     = errors.New("refresh token tidak valid")
	ErrRefreshExpired     = errors.New("sesi sudah berakhir, silakan login ulang")
	ErrRefreshReused      = errors.New("sesi dicabut karena terdeteksi penggunaan token berulang")
)

type UserFinder interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id int64) (*models.User, error)
}

type RefreshTokenStore interface {
	Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	FindByHash(ctx context.Context, tokenHash string) (*repository.RefreshToken, error)
	MarkUsed(ctx context.Context, id int64) error
	Revoke(ctx context.Context, id int64) error
	RevokeAllForUser(ctx context.Context, userID int64) error
}

type AuthService struct {
	userRepo    UserFinder
	refreshRepo RefreshTokenStore
	jwtSecret   string
}

func NewAuthService(userRepo UserFinder, refreshRepo RefreshTokenStore, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		jwtSecret:   jwtSecret,
	}
}

type LoginResult struct {
	Token        string
	RefreshToken string
	Role         string
	ExpiresIn    int // detik
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokens(ctx, user)
}

// issueTokens menerbitkan sepasang token baru dan menyimpan refresh-nya.
func (s *AuthService) issueTokens(ctx context.Context, user *models.User) (*LoginResult, error) {
	accessToken, err := utils.GenerateToken(user.ID, user.Role, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(utils.RefreshTokenTTL)
	if err := s.refreshRepo.Create(ctx, user.ID, utils.HashToken(refreshToken), expiresAt); err != nil {
		return nil, err
	}

	return &LoginResult{
		Token:        accessToken,
		RefreshToken: refreshToken,
		Role:         user.Role,
		ExpiresIn:    int(utils.AccessTokenTTL.Seconds()),
	}, nil
}

// Refresh menukar refresh token dengan sepasang token baru.
// Kalau token yang dipakai sudah pernah ditukar atau dicabut,
// seluruh sesi user dicabut karena token diduga bocor.
func (s *AuthService) Refresh(ctx context.Context, token string) (*LoginResult, error) {
	stored, err := s.refreshRepo.FindByHash(ctx, utils.HashToken(token))
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			return nil, ErrInvalidRefresh
		}
		return nil, err
	}

	// Pemakaian ulang: token ini sudah ditukar atau dicabut sebelumnya.
	if stored.UsedAt != nil || stored.RevokedAt != nil {
		if err := s.refreshRepo.RevokeAllForUser(ctx, stored.UserID); err != nil {
			return nil, err
		}
		return nil, ErrRefreshReused
	}

	if time.Now().After(stored.ExpiresAt) {
		return nil, ErrRefreshExpired
	}

	user, err := s.userRepo.FindByID(ctx, stored.UserID)
	if err != nil {
		return nil, ErrInvalidRefresh
	}

	if err := s.refreshRepo.MarkUsed(ctx, stored.ID); err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user)
}

// Logout mencabut satu refresh token.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	stored, err := s.refreshRepo.FindByHash(ctx, utils.HashToken(token))
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			return ErrInvalidRefresh
		}
		return err
	}
	return s.refreshRepo.Revoke(ctx, stored.ID)
}

// Me mengembalikan data user pemilik token yang sedang dipakai.
func (s *AuthService) Me(ctx context.Context, userID int64) (*models.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}