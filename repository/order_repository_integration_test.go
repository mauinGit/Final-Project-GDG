//go:build integration

package repository

import (
	"context"
	"testing"

	"FinalProjectBE/models"
)

// buatMenuUntukOrder menyisipkan satu menu dan mengembalikannya.
func buatMenuUntukOrder(t *testing.T, repo *MenuRepository, nama string, harga int) *models.MenuItem {
	t.Helper()

	item := &models.MenuItem{Name: nama, Price: harga, Category: "makanan"}
	if err := repo.Create(context.Background(), item); err != nil {
		t.Fatalf("gagal menyiapkan menu: %v", err)
	}
	return item
}

// orderContoh menyusun order sederhana dengan satu item.
func orderContoh(menu *models.MenuItem, qty int, queue int) *models.Order {
	subtotal := menu.Price * qty
	return &models.Order{
		QueueNumber:   queue,
		CustomerName:  "Budi",
		Status:        models.StatusPending,
		Subtotal:      subtotal,
		Total:         subtotal,
		PaymentMethod: models.PaymentCash,
		AmountPaid:    subtotal,
		Items: []models.OrderItem{
			{MenuItemID: menu.ID, MenuName: menu.Name, Quantity: qty, PriceAtOrder: menu.Price},
		},
	}
}

func TestOrderRepository_CreateMenyimpanOrderDanItem(t *testing.T) {
	pool := setupTestDB(t)
	menuRepo := NewMenuRepository(pool)
	orderRepo := NewOrderRepository(pool)
	ctx := context.Background()

	menu := buatMenuUntukOrder(t, menuRepo, "Nasi Goreng", 25000)
	order := orderContoh(menu, 2, 1)
	order.Subtotal = 50000
	order.Total = 50000
	order.AmountPaid = 50000

	saved, err := orderRepo.Create(ctx, order)
	if err != nil {
		t.Fatalf("gagal menyimpan order: %v", err)
	}
	if saved.ID == 0 {
		t.Fatal("id order harusnya terisi")
	}
	if saved.Items[0].ID == 0 {
		t.Error("id item harusnya terisi")
	}

	found, err := orderRepo.FindByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("gagal mengambil order: %v", err)
	}
	if len(found.Items) != 1 {
		t.Fatalf("harusnya 1 item, dapat %d", len(found.Items))
	}
	if found.Items[0].PriceAtOrder != 25000 {
		t.Errorf("harga snapshot salah: %d", found.Items[0].PriceAtOrder)
	}
	if found.Total != 50000 {
		t.Errorf("total salah: %d", found.Total)
	}
}

// Kalau item merujuk menu yang tidak ada, seluruh transaksi harus batal.
func TestOrderRepository_RollbackSaatItemGagal(t *testing.T) {
	pool := setupTestDB(t)
	orderRepo := NewOrderRepository(pool)
	ctx := context.Background()

	order := &models.Order{
		QueueNumber:   1,
		CustomerName:  "Budi",
		Status:        models.StatusPending,
		PaymentMethod: models.PaymentCash,
		Items: []models.OrderItem{
			{MenuItemID: 9999, MenuName: "Hantu", Quantity: 1, PriceAtOrder: 1000},
		},
	}

	if _, err := orderRepo.Create(ctx, order); err == nil {
		t.Fatal("harusnya gagal karena menu_item_id tidak ada")
	}

	// Order induk tidak boleh tertinggal di database.
	var jumlah int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM orders`).Scan(&jumlah); err != nil {
		t.Fatalf("gagal menghitung order: %v", err)
	}
	if jumlah != 0 {
		t.Errorf("rollback gagal, masih ada %d order tersimpan", jumlah)
	}
}

func TestOrderRepository_CountOrdersToday(t *testing.T) {
	pool := setupTestDB(t)
	menuRepo := NewMenuRepository(pool)
	orderRepo := NewOrderRepository(pool)
	ctx := context.Background()

	menu := buatMenuUntukOrder(t, menuRepo, "Es Teh", 5000)

	awal, err := orderRepo.CountOrdersToday(ctx)
	if err != nil {
		t.Fatalf("gagal menghitung: %v", err)
	}
	if awal != 0 {
		t.Errorf("database baru harusnya 0 order, dapat %d", awal)
	}

	orderRepo.Create(ctx, orderContoh(menu, 1, 1))
	orderRepo.Create(ctx, orderContoh(menu, 1, 2))

	akhir, err := orderRepo.CountOrdersToday(ctx)
	if err != nil {
		t.Fatalf("gagal menghitung: %v", err)
	}
	if akhir != 2 {
		t.Errorf("harusnya 2 order hari ini, dapat %d", akhir)
	}
}

func TestOrderRepository_PaginationDanTotalItem(t *testing.T) {
	pool := setupTestDB(t)
	menuRepo := NewMenuRepository(pool)
	orderRepo := NewOrderRepository(pool)
	ctx := context.Background()

	menu := buatMenuUntukOrder(t, menuRepo, "Mie Goreng", 20000)
	for i := 1; i <= 5; i++ {
		orderRepo.Create(ctx, orderContoh(menu, 1, i))
	}

	hal1, total, err := orderRepo.FindAll(ctx, OrderFilter{Page: 1, Limit: 2})
	if err != nil {
		t.Fatalf("gagal mengambil halaman 1: %v", err)
	}
	if total != 5 {
		t.Errorf("total item salah: mau 5, dapat %d", total)
	}
	if len(hal1) != 2 {
		t.Errorf("halaman 1 harusnya 2 baris, dapat %d", len(hal1))
	}

	hal2, _, err := orderRepo.FindAll(ctx, OrderFilter{Page: 2, Limit: 2})
	if err != nil {
		t.Fatalf("gagal mengambil halaman 2: %v", err)
	}
	if len(hal2) != 2 {
		t.Errorf("halaman 2 harusnya 2 baris, dapat %d", len(hal2))
	}

	// Tidak boleh ada ID yang muncul di dua halaman.
	for _, a := range hal1 {
		for _, b := range hal2 {
			if a.ID == b.ID {
				t.Errorf("order id %d muncul di dua halaman", a.ID)
			}
		}
	}
}

// COUNT harus ikut kena filter, bukan menghitung seluruh tabel.
func TestOrderRepository_FilterStatusIkutMempengaruhiTotal(t *testing.T) {
	pool := setupTestDB(t)
	menuRepo := NewMenuRepository(pool)
	orderRepo := NewOrderRepository(pool)
	ctx := context.Background()

	menu := buatMenuUntukOrder(t, menuRepo, "Soto", 18000)
	o1, _ := orderRepo.Create(ctx, orderContoh(menu, 1, 1))
	orderRepo.Create(ctx, orderContoh(menu, 1, 2))
	orderRepo.Create(ctx, orderContoh(menu, 1, 3))

	// Satu order diubah jadi cooking.
	if _, err := orderRepo.UpdateStatus(ctx, o1.ID, models.StatusCooking); err != nil {
		t.Fatalf("gagal mengubah status: %v", err)
	}

	_, totalPending, err := orderRepo.FindAll(ctx, OrderFilter{
		Status: models.StatusPending, Page: 1, Limit: 10,
	})
	if err != nil {
		t.Fatalf("gagal memfilter: %v", err)
	}
	if totalPending != 2 {
		t.Errorf("total pending salah: mau 2, dapat %d", totalPending)
	}
}