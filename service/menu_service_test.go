package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"FinalProjectBE/models"
)

type mockMenuStore struct {
	createErr error
	updateErr error
	deleteErr error

	findAllResp []models.MenuItem
	findAllErr  error

	findByIDResp *models.MenuItem
	findByIDErr  error

	lastCreated  *models.MenuItem
	lastCategory string
}

func (m *mockMenuStore) Create(ctx context.Context, item *models.MenuItem) error {
	if m.createErr != nil {
		return m.createErr
	}
	item.ID = 1
	m.lastCreated = item
	return nil
}

func (m *mockMenuStore) FindAll(ctx context.Context, category string) ([]models.MenuItem, error) {
	m.lastCategory = category
	return m.findAllResp, m.findAllErr
}

func (m *mockMenuStore) FindByID(ctx context.Context, id int64) (*models.MenuItem, error) {
	return m.findByIDResp, m.findByIDErr
}

func (m *mockMenuStore) Update(ctx context.Context, item *models.MenuItem) error {
	return m.updateErr
}

func (m *mockMenuStore) Delete(ctx context.Context, id int64) error {
	return m.deleteErr
}

// Error PostgreSQL palsu untuk menguji penerjemahan kode error.
func pgErr(code string) error {
	return &pgconn.PgError{Code: code}
}

// Create

func TestMenuCreate_Berhasil(t *testing.T) {
	repo := &mockMenuStore{}
	svc := NewMenuService(repo)

	item := &models.MenuItem{Name: "Nasi Goreng", Price: 20000, Category: "makanan"}
	if err := svc.Create(context.Background(), item); err != nil {
		t.Fatalf("harusnya berhasil, dapat: %v", err)
	}
	if item.ID != 1 {
		t.Errorf("id harusnya terisi, dapat %d", item.ID)
	}
}

func TestMenuCreate_NamaDipangkasSpasi(t *testing.T) {
	repo := &mockMenuStore{}
	svc := NewMenuService(repo)

	item := &models.MenuItem{Name: "  Es Teh  ", Price: 5000, Category: " minuman "}
	if err := svc.Create(context.Background(), item); err != nil {
		t.Fatalf("harusnya berhasil, dapat: %v", err)
	}
	if item.Name != "Es Teh" {
		t.Errorf("nama harus dipangkas, dapat %q", item.Name)
	}
	if item.Category != "minuman" {
		t.Errorf("kategori harus dipangkas, dapat %q", item.Category)
	}
}

func TestMenuCreate_NamaKosong(t *testing.T) {
	svc := NewMenuService(&mockMenuStore{})

	err := svc.Create(context.Background(), &models.MenuItem{
		Name: "   ", Price: 1000, Category: "makanan",
	})
	if !errors.Is(err, ErrMenuNameRequired) {
		t.Errorf("harusnya ErrMenuNameRequired, dapat: %v", err)
	}
}

func TestMenuCreate_KategoriKosong(t *testing.T) {
	svc := NewMenuService(&mockMenuStore{})

	err := svc.Create(context.Background(), &models.MenuItem{
		Name: "Nasi", Price: 1000, Category: "",
	})
	if !errors.Is(err, ErrMenuCategoryEmpty) {
		t.Errorf("harusnya ErrMenuCategoryEmpty, dapat: %v", err)
	}
}

func TestMenuCreate_HargaNegatif(t *testing.T) {
	svc := NewMenuService(&mockMenuStore{})

	err := svc.Create(context.Background(), &models.MenuItem{
		Name: "Nasi", Price: -5000, Category: "makanan",
	})
	if !errors.Is(err, ErrMenuPriceInvalid) {
		t.Errorf("harusnya ErrMenuPriceInvalid, dapat: %v", err)
	}
}

func TestMenuCreate_NamaDuplikat(t *testing.T) {
	repo := &mockMenuStore{createErr: pgErr("23505")}
	svc := NewMenuService(repo)

	err := svc.Create(context.Background(), &models.MenuItem{
		Name: "Nasi Goreng", Price: 20000, Category: "makanan",
	})
	if !errors.Is(err, ErrMenuNameTaken) {
		t.Errorf("harusnya ErrMenuNameTaken, dapat: %v", err)
	}
}

// Update

func TestMenuUpdate_NamaDuplikat(t *testing.T) {
	repo := &mockMenuStore{updateErr: pgErr("23505")}
	svc := NewMenuService(repo)

	err := svc.Update(context.Background(), &models.MenuItem{
		ID: 1, Name: "Nasi Goreng", Price: 20000, Category: "makanan",
	})
	if !errors.Is(err, ErrMenuNameTaken) {
		t.Errorf("harusnya ErrMenuNameTaken, dapat: %v", err)
	}
}

func TestMenuUpdate_ValidasiTetapBerlaku(t *testing.T) {
	svc := NewMenuService(&mockMenuStore{})

	err := svc.Update(context.Background(), &models.MenuItem{
		ID: 1, Name: "Nasi", Price: -1, Category: "makanan",
	})
	if !errors.Is(err, ErrMenuPriceInvalid) {
		t.Errorf("harusnya ErrMenuPriceInvalid, dapat: %v", err)
	}
}

// Delete

func TestMenuDelete_Berhasil(t *testing.T) {
	svc := NewMenuService(&mockMenuStore{})

	if err := svc.Delete(context.Background(), 1); err != nil {
		t.Errorf("harusnya berhasil, dapat: %v", err)
	}
}

func TestMenuDelete_MasihDipakai(t *testing.T) {
	repo := &mockMenuStore{deleteErr: pgErr("23503")}
	svc := NewMenuService(repo)

	err := svc.Delete(context.Background(), 1)
	if !errors.Is(err, ErrMenuInUse) {
		t.Errorf("harusnya ErrMenuInUse, dapat: %v", err)
	}
}

// List

func TestMenuList_KategoriDipangkas(t *testing.T) {
	repo := &mockMenuStore{findAllResp: []models.MenuItem{{ID: 1}}}
	svc := NewMenuService(repo)

	items, err := svc.List(context.Background(), "  minuman  ")
	if err != nil {
		t.Fatalf("harusnya berhasil, dapat: %v", err)
	}
	if repo.lastCategory != "minuman" {
		t.Errorf("kategori harus dipangkas, dapat %q", repo.lastCategory)
	}
	if len(items) != 1 {
		t.Errorf("mau 1 item, dapat %d", len(items))
	}
}