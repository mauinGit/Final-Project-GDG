//go:build integration

package repository

import (
	"context"
	"testing"

	"FinalProjectBE/models"
)

func TestMenuRepository_CreateDanFindByID(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewMenuRepository(pool)
	ctx := context.Background()

	item := &models.MenuItem{
		Name:     "Nasi Goreng",
		Price:    25000,
		Category: "makanan",
	}

	if err := repo.Create(ctx, item); err != nil {
		t.Fatalf("gagal menyimpan menu: %v", err)
	}
	if item.ID == 0 {
		t.Fatal("id harusnya terisi setelah insert")
	}
	if item.CreatedAt.IsZero() {
		t.Error("created_at harusnya diisi database")
	}

	found, err := repo.FindByID(ctx, item.ID)
	if err != nil {
		t.Fatalf("gagal mengambil menu: %v", err)
	}
	if found.Name != "Nasi Goreng" || found.Price != 25000 {
		t.Errorf("data tidak cocok: %+v", found)
	}
}

func TestMenuRepository_FindByID_TidakAda(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewMenuRepository(pool)

	_, err := repo.FindByID(context.Background(), 9999)
	if err != ErrMenuItemNotFound {
		t.Errorf("harusnya ErrMenuItemNotFound, dapat: %v", err)
	}
}

// Menguji index unik case-insensitive dari migrasi 003.
func TestMenuRepository_NamaUnikTanpaPeduliHuruf(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewMenuRepository(pool)
	ctx := context.Background()

	first := &models.MenuItem{Name: "Kwetiau Goreng", Price: 20000, Category: "makanan"}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("insert pertama harusnya berhasil: %v", err)
	}

	second := &models.MenuItem{Name: "kwetiau goreng", Price: 22000, Category: "makanan"}
	err := repo.Create(ctx, second)
	if err == nil {
		t.Fatal("nama beda huruf besar-kecil harusnya ditolak")
	}
	if !isUniqueViolationErr(err) {
		t.Errorf("harusnya pelanggaran unique (23505), dapat: %v", err)
	}
}

// Menguji ON DELETE RESTRICT: menu yang sudah dipesan tidak boleh dihapus.
func TestMenuRepository_TidakBisaHapusMenuYangSudahDipesan(t *testing.T) {
	pool := setupTestDB(t)
	menuRepo := NewMenuRepository(pool)
	orderRepo := NewOrderRepository(pool)
	ctx := context.Background()

	menu := &models.MenuItem{Name: "Es Teh", Price: 5000, Category: "minuman"}
	if err := menuRepo.Create(ctx, menu); err != nil {
		t.Fatalf("gagal menyimpan menu: %v", err)
	}

	order := &models.Order{
		QueueNumber:   1,
		CustomerName:  "Budi",
		Status:        models.StatusPending,
		Subtotal:      5000,
		Total:         5000,
		PaymentMethod: models.PaymentCash,
		AmountPaid:    5000,
		Items: []models.OrderItem{
			{MenuItemID: menu.ID, MenuName: menu.Name, Quantity: 1, PriceAtOrder: menu.Price},
		},
	}
	if _, err := orderRepo.Create(ctx, order); err != nil {
		t.Fatalf("gagal menyimpan order: %v", err)
	}

	err := menuRepo.Delete(ctx, menu.ID)
	if err == nil {
		t.Fatal("menu yang sudah dipesan harusnya tidak bisa dihapus")
	}
	if !isForeignKeyViolationErr(err) {
		t.Errorf("harusnya pelanggaran foreign key (23503), dapat: %v", err)
	}
}

func TestMenuRepository_FilterKategori(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewMenuRepository(pool)
	ctx := context.Background()

	repo.Create(ctx, &models.MenuItem{Name: "Nasi Goreng", Price: 25000, Category: "makanan"})
	repo.Create(ctx, &models.MenuItem{Name: "Mie Goreng", Price: 23000, Category: "makanan"})
	repo.Create(ctx, &models.MenuItem{Name: "Es Teh", Price: 5000, Category: "minuman"})

	semua, err := repo.FindAll(ctx, "")
	if err != nil {
		t.Fatalf("gagal mengambil semua menu: %v", err)
	}
	if len(semua) != 3 {
		t.Errorf("tanpa filter harusnya 3 menu, dapat %d", len(semua))
	}

	minuman, err := repo.FindAll(ctx, "minuman")
	if err != nil {
		t.Fatalf("gagal memfilter menu: %v", err)
	}
	if len(minuman) != 1 {
		t.Errorf("kategori minuman harusnya 1 menu, dapat %d", len(minuman))
	}
}