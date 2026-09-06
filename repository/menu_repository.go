package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"FinalProjectBE/models"
)

var ErrMenuItemNotFound = errors.New("menu tidak ditemukan")

type MenuRepository struct {
	pool *pgxpool.Pool
}

func NewMenuRepository(pool *pgxpool.Pool) *MenuRepository {
	return &MenuRepository{pool: pool}
}

func (r *MenuRepository) Create(ctx context.Context, m *models.MenuItem) error {
	query := `
		INSERT INTO menu_items (name, price, category, image_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query, m.Name, m.Price, m.Category, m.ImageURL).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("gagal menyimpan menu: %w", err)
	}
	return nil
}

func (r *MenuRepository) FindAll(ctx context.Context, category string) ([]models.MenuItem, error) {
	query := `
		SELECT id, name, price, category, image_url, created_at, updated_at
		FROM menu_items
		WHERE ($1 = '' OR category = $1)
		ORDER BY category, name
	`
	rows, err := r.pool.Query(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar menu: %w", err)
	}
	defer rows.Close()

	items := make([]models.MenuItem, 0)
	for rows.Next() {
		var m models.MenuItem
		if err := rows.Scan(&m.ID, &m.Name, &m.Price, &m.Category,
			&m.ImageURL, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("gagal membaca menu: %w", err)
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

func (r *MenuRepository) FindByID(ctx context.Context, id int64) (*models.MenuItem, error) {
	query := `
		SELECT id, name, price, category, image_url, created_at, updated_at
		FROM menu_items WHERE id = $1
	`
	var m models.MenuItem
	err := r.pool.QueryRow(ctx, query, id).
		Scan(&m.ID, &m.Name, &m.Price, &m.Category,
			&m.ImageURL, &m.CreatedAt, &m.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMenuItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil menu: %w", err)
	}
	return &m, nil
}

func (r *MenuRepository) Update(ctx context.Context, m *models.MenuItem) error {
	query := `
		UPDATE menu_items
		SET name = $1, price = $2, category = $3, image_url = $4, updated_at = now()
		WHERE id = $5
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query, m.Name, m.Price, m.Category, m.ImageURL, m.ID).
		Scan(&m.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrMenuItemNotFound
	}
	if err != nil {
		return fmt.Errorf("gagal memperbarui menu: %w", err)
	}
	return nil
}

func (r *MenuRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM menu_items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("gagal menghapus menu: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMenuItemNotFound
	}
	return nil
}