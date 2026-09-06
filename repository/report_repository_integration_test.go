//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"FinalProjectBE/models"
)

// orderLengkap menyusun order dengan kendali penuh atas angka & metode bayar.
func orderLengkap(menu *models.MenuItem, qty, diskon int, metode string, queue int) *models.Order {
	subtotal := menu.Price * qty
	total := subtotal - diskon

	o := &models.Order{
		QueueNumber:   queue,
		CustomerName:  "Budi",
		Status:        models.StatusPending,
		Subtotal:      subtotal,
		Discount:      diskon,
		Total:         total,
		PaymentMethod: metode,
		Items: []models.OrderItem{
			{MenuItemID: menu.ID, MenuName: menu.Name, Quantity: qty, PriceAtOrder: menu.Price},
		},
	}
	if metode == models.PaymentCash {
		o.AmountPaid = total
	}
	return o
}

func TestReportRepository_RingkasanMengabaikanOrderDibatalkan(t *testing.T) {
	pool := setupTestDB(t)
	menuRepo := NewMenuRepository(pool)
	orderRepo := NewOrderRepository(pool)
	reportRepo := NewReportRepository(pool)
	ctx := context.Background()

	menu := buatMenuUntukOrder(t, menuRepo, "Nasi Goreng", 20000)

	// Dua order normal.
	orderRepo.Create(ctx, orderLengkap(menu, 1, 0, models.PaymentCash, 1))
	orderRepo.Create(ctx, orderLengkap(menu, 1, 0, models.PaymentCash, 2))

	// Satu order yang kemudian dibatalkan.
	batal, _ := orderRepo.Create(ctx, orderLengkap(menu, 5, 0, models.PaymentCash, 3))
	if _, err := orderRepo.UpdateStatus(ctx, batal.ID, models.StatusCancelled); err != nil {
		t.Fatalf("gagal membatalkan order: %v", err)
	}

	rep, err := reportRepo.DailySummary(ctx, time.Now())
	if err != nil {
		t.Fatalf("gagal menyusun ringkasan: %v", err)
	}

	if rep.TotalOrders != 3 {
		t.Errorf("total order harusnya 3 (termasuk yang batal), dapat %d", rep.TotalOrders)
	}
	if rep.CancelledOrders != 1 {
		t.Errorf("order dibatalkan harusnya 1, dapat %d", rep.CancelledOrders)
	}
	// 2 x 20000, order yang batal (100000) tidak dihitung.
	if rep.NetRevenue != 40000 {
		t.Errorf("omzet harusnya 40000, dapat %d", rep.NetRevenue)
	}
	if rep.GrossRevenue != 40000 {
		t.Errorf("gross harusnya 40000, dapat %d", rep.GrossRevenue)
	}
}

func TestReportRepository_PisahTunaiDanNonTunai(t *testing.T) {
	pool := setupTestDB(t)
	menuRepo := NewMenuRepository(pool)
	orderRepo := NewOrderRepository(pool)
	reportRepo := NewReportRepository(pool)
	ctx := context.Background()

	menu := buatMenuUntukOrder(t, menuRepo, "Es Teh", 10000)

	orderRepo.Create(ctx, orderLengkap(menu, 2, 0, models.PaymentCash, 1))    // 20000 tunai
	orderRepo.Create(ctx, orderLengkap(menu, 3, 0, models.PaymentNonCash, 2)) // 30000 non-tunai

	rep, err := reportRepo.DailySummary(ctx, time.Now())
	if err != nil {
		t.Fatalf("gagal menyusun ringkasan: %v", err)
	}

	if rep.CashRevenue != 20000 {
		t.Errorf("omzet tunai salah: mau 20000, dapat %d", rep.CashRevenue)
	}
	if rep.NonCashRevenue != 30000 {
		t.Errorf("omzet non-tunai salah: mau 30000, dapat %d", rep.NonCashRevenue)
	}
	if rep.CashRevenue+rep.NonCashRevenue != rep.NetRevenue {
		t.Errorf("tunai + non-tunai harus sama dengan net: %d + %d != %d",
			rep.CashRevenue, rep.NonCashRevenue, rep.NetRevenue)
	}
}

func TestReportRepository_DiskonMengurangiNetBukanGross(t *testing.T) {
	pool := setupTestDB(t)
	menuRepo := NewMenuRepository(pool)
	orderRepo := NewOrderRepository(pool)
	reportRepo := NewReportRepository(pool)
	ctx := context.Background()

	menu := buatMenuUntukOrder(t, menuRepo, "Ayam Bakar", 30000)
	orderRepo.Create(ctx, orderLengkap(menu, 1, 5000, models.PaymentCash, 1))

	rep, err := reportRepo.DailySummary(ctx, time.Now())
	if err != nil {
		t.Fatalf("gagal menyusun ringkasan: %v", err)
	}

	if rep.GrossRevenue != 30000 {
		t.Errorf("gross harusnya sebelum diskon (30000), dapat %d", rep.GrossRevenue)
	}
	if rep.TotalDiscount != 5000 {
		t.Errorf("total diskon salah: mau 5000, dapat %d", rep.TotalDiscount)
	}
	if rep.NetRevenue != 25000 {
		t.Errorf("net harusnya setelah diskon (25000), dapat %d", rep.NetRevenue)
	}
}

func TestReportRepository_MenuTerlarisTerurut(t *testing.T) {
	pool := setupTestDB(t)
	menuRepo := NewMenuRepository(pool)
	orderRepo := NewOrderRepository(pool)
	reportRepo := NewReportRepository(pool)
	ctx := context.Background()

	nasi := buatMenuUntukOrder(t, menuRepo, "Nasi Goreng", 20000)
	teh := buatMenuUntukOrder(t, menuRepo, "Es Teh", 5000)

	// Nasi 3 porsi, teh 7 porsi.
	orderRepo.Create(ctx, orderLengkap(nasi, 3, 0, models.PaymentCash, 1))
	orderRepo.Create(ctx, orderLengkap(teh, 7, 0, models.PaymentCash, 2))

	items, err := reportRepo.TopItems(ctx, time.Now(), 5)
	if err != nil {
		t.Fatalf("gagal mengambil menu terlaris: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("harusnya 2 menu, dapat %d", len(items))
	}

	// Teh harus di urutan pertama karena kuantitasnya lebih banyak.
	if items[0].MenuName != "Es Teh" {
		t.Errorf("urutan salah, teratas: %s", items[0].MenuName)
	}
	if items[0].QuantitySold != 7 {
		t.Errorf("kuantitas teh salah: mau 7, dapat %d", items[0].QuantitySold)
	}
	if items[0].Revenue != 35000 {
		t.Errorf("omzet teh salah: mau 35000, dapat %d", items[0].Revenue)
	}
}

func TestReportRepository_MenuTerlarisAbaikanOrderDibatalkan(t *testing.T) {
	pool := setupTestDB(t)
	menuRepo := NewMenuRepository(pool)
	orderRepo := NewOrderRepository(pool)
	reportRepo := NewReportRepository(pool)
	ctx := context.Background()

	menu := buatMenuUntukOrder(t, menuRepo, "Sate", 25000)

	batal, _ := orderRepo.Create(ctx, orderLengkap(menu, 10, 0, models.PaymentCash, 1))
	orderRepo.UpdateStatus(ctx, batal.ID, models.StatusCancelled)

	items, err := reportRepo.TopItems(ctx, time.Now(), 5)
	if err != nil {
		t.Fatalf("gagal mengambil menu terlaris: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("order dibatalkan tidak boleh masuk terlaris, dapat %d baris", len(items))
	}
}

func TestReportRepository_TanggalKosongMengembalikanNol(t *testing.T) {
	pool := setupTestDB(t)
	reportRepo := NewReportRepository(pool)
	ctx := context.Background()

	kemarin := time.Now().AddDate(0, 0, -1)
	rep, err := reportRepo.DailySummary(ctx, kemarin)
	if err != nil {
		t.Fatalf("tanggal kosong tidak boleh error: %v", err)
	}
	if rep.TotalOrders != 0 || rep.NetRevenue != 0 {
		t.Errorf("tanggal kosong harusnya nol semua: %+v", rep)
	}

	items, err := reportRepo.TopItems(ctx, kemarin, 5)
	if err != nil {
		t.Fatalf("tanggal kosong tidak boleh error: %v", err)
	}
	if items == nil {
		t.Error("harusnya slice kosong, bukan nil")
	}
}