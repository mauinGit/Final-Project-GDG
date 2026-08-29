package service

import (
	"context"
	"errors"

	"FinalProjectBE/models"
	"FinalProjectBE/repository"
	"FinalProjectBE/utils"
)

var ErrInvalidCredentials = errors.New("email atau password salah")

type UserFinder interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)
}

type AuthService struct {
	userRepo  UserFinder
	jwtSecret string
}

func NewAuthService(userRepo UserFinder, jwtSecret string) *AuthService {
	return &AuthService{userRepo: userRepo, jwtSecret: jwtSecret}
}

type LoginResult struct {
	Token string
	Role  string
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

	token, err := utils.GenerateToken(user.ID, user.Role, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &LoginResult{Token: token, Role: user.Role}, nil
}