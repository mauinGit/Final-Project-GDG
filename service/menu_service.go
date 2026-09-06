package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"FinalProjectBE/models"
	"FinalProjectBE/repository"
)

var (
	ErrMenuNameRequired  = errors.New("nama menu wajib diisi")
	ErrMenuPriceInvalid  = errors.New("harga tidak boleh negatif")
	ErrMenuCategoryEmpty = errors.New("kategori wajib diisi")
	ErrMenuNameTaken     = errors.New("nama menu sudah dipakai")
	ErrMenuInUse         = errors.New("menu tidak bisa dihapus karena sudah pernah dipesan")
)

type MenuService struct {
	repo *repository.MenuRepository
}

func NewMenuService(repo *repository.MenuRepository) *MenuService {
	return &MenuService{repo: repo}
}

func (s *MenuService) validate(m *models.MenuItem) error {
	m.Name = strings.TrimSpace(m.Name)
	m.Category = strings.TrimSpace(m.Category)

	if m.Name == "" {
		return ErrMenuNameRequired
	}
	if m.Category == "" {
		return ErrMenuCategoryEmpty
	}
	if m.Price < 0 {
		return ErrMenuPriceInvalid
	}
	return nil
}

func (s *MenuService) Create(ctx context.Context, m *models.MenuItem) error {
	if err := s.validate(m); err != nil {
		return err
	}

	if err := s.repo.Create(ctx, m); err != nil {
		if isUniqueViolation(err) {
			return ErrMenuNameTaken
		}
		return err
	}
	return nil
}

func (s *MenuService) List(ctx context.Context, category string) ([]models.MenuItem, error) {
	return s.repo.FindAll(ctx, strings.TrimSpace(category))
}

func (s *MenuService) GetByID(ctx context.Context, id int64) (*models.MenuItem, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *MenuService) Update(ctx context.Context, m *models.MenuItem) error {
	if err := s.validate(m); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, m); err != nil {
		if isUniqueViolation(err) {
			return ErrMenuNameTaken
		}
		return err
	}
	return nil
}

func (s *MenuService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if isForeignKeyViolation(err) {
			return ErrMenuInUse
		}
		return err
	}
	return nil
}

// Kode error PostgreSQL: 23505 = unique_violation, 23503 = foreign_key_violation.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

var _ = fmt.Sprintf // hapus baris ini kalau fmt tidak terpakai