package service

import (
	"context"
	"errors"
	"testing"

	"FinalProjectBE/models"
)
type mockOrderStore struct {
	countToday   int
	countErr     error
	created      *models.Order
	createErr    error
	findByIDResp *models.Order
	findByIDErr  error
	updateResp   *models.Order
	updateErr    error
	findAllResp  []models.Order
	findAllErr   error

	menuPrices    map[int64]models.MenuItem
	menuPricesErr error

	lastUpdateStatus string
}

func (m *mockOrderStore) Create(ctx context.Context, order *models.Order) (*models.Order, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	order.ID = 99
	m.created = order
	return order, nil
}

func (m *mockOrderStore) FindAll(ctx context.Context, statusFilter string) ([]models.Order, error) {
	return m.findAllResp, m.findAllErr
}

func (m *mockOrderStore) FindByID(ctx context.Context, id int64) (*models.Order, error) {
	return m.findByIDResp, m.findByIDErr
}

func (m *mockOrderStore) UpdateStatus(ctx context.Context, id int64, status string) (*models.Order, error) {
	m.lastUpdateStatus = status
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	if m.updateResp != nil {
		return m.updateResp, nil
	}
	return &models.Order{ID: id, Status: status}, nil
}

func (m *mockOrderStore) CountOrdersToday(ctx context.Context) (int, error) {
	return m.countToday, m.countErr
}

func (m *mockOrderStore) FindMenuPrices(ctx context.Context, ids []int64) (map[int64]models.MenuItem, error) {
	if m.menuPricesErr != nil {
		return nil, m.menuPricesErr
	}
	if m.menuPrices == nil {
		return map[int64]models.MenuItem{}, nil
	}
	return m.menuPrices, nil
}

// Create Order 

func TestCreateOrder_Berhasil(t *testing.T) {
	repo := &mockOrderStore{countToday: 4, menuPrices: menuContoh()}
	svc := NewOrderService(repo)

	order, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerName: "Budi",
		Items: []models.OrderItem{
			{MenuItemID: 1, Quantity: 2},
			{MenuItemID: 2, Quantity: 1},
		},
		PaymentMethod: models.PaymentCash,
		AmountPaid:    50000,
	})

	if err != nil {
		t.Fatalf("harusnya berhasil, dapat error: %v", err)
	}
	if order.QueueNumber != 5 {
		t.Errorf("nomor urut salah: mau 5 (4+1), dapat %d", order.QueueNumber)
	}
	if order.Status != models.StatusPending {
		t.Errorf("status awal harus pending, dapat %s", order.Status)
	}
	// 20000*2 + 5000*1
	if order.Subtotal != 45000 {
		t.Errorf("subtotal salah: mau 45000, dapat %d", order.Subtotal)
	}
	if order.Total != 45000 {
		t.Errorf("total salah: mau 45000, dapat %d", order.Total)
	}
	if order.ChangeAmount != 5000 {
		t.Errorf("kembalian salah: mau 5000, dapat %d", order.ChangeAmount)
	}
}

func TestCreateOrder_SnapshotHargaDariMenu(t *testing.T) {
	repo := &mockOrderStore{menuPrices: menuContoh()}
	svc := NewOrderService(repo)

	// Kasir mengirim harga & nama palsu — harus diabaikan.
	order, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerName: "Budi",
		Items: []models.OrderItem{
			{MenuItemID: 1, Quantity: 1, MenuName: "Gratisan", PriceAtOrder: 1},
		},
		PaymentMethod: models.PaymentNonCash,
	})

	if err != nil {
		t.Fatalf("harusnya berhasil, dapat: %v", err)
	}
	if order.Items[0].PriceAtOrder != 20000 {
		t.Errorf("harga harus dari menu (20000), dapat %d", order.Items[0].PriceAtOrder)
	}
	if order.Items[0].MenuName != "Nasi Goreng" {
		t.Errorf("nama harus dari menu, dapat %s", order.Items[0].MenuName)
	}
}

func TestCreateOrder_NonCashTidakAdaKembalian(t *testing.T) {
	repo := &mockOrderStore{menuPrices: menuContoh()}
	svc := NewOrderService(repo)

	order, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerName:  "Budi",
		Items:         []models.OrderItem{{MenuItemID: 1, Quantity: 1}},
		PaymentMethod: models.PaymentNonCash,
		AmountPaid:    999999, // harus diabaikan
	})

	if err != nil {
		t.Fatalf("harusnya berhasil, dapat: %v", err)
	}
	if order.AmountPaid != 0 || order.ChangeAmount != 0 {
		t.Errorf("non-cash tidak boleh mencatat uang: paid=%d change=%d",
			order.AmountPaid, order.ChangeAmount)
	}
}

func TestCreateOrder_UangKurang(t *testing.T) {
	repo := &mockOrderStore{menuPrices: menuContoh()}
	svc := NewOrderService(repo)

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerName:  "Budi",
		Items:         []models.OrderItem{{MenuItemID: 1, Quantity: 1}},
		PaymentMethod: models.PaymentCash,
		AmountPaid:    10000,
	})
	if !errors.Is(err, ErrInsufficientPaid) {
		t.Errorf("harusnya ErrInsufficientPaid, dapat: %v", err)
	}
}

func TestCreateOrder_DiskonMelebihiSubtotal(t *testing.T) {
	repo := &mockOrderStore{menuPrices: menuContoh()}
	svc := NewOrderService(repo)

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerName:  "Budi",
		Items:         []models.OrderItem{{MenuItemID: 1, Quantity: 1}},
		Discount:      99999,
		PaymentMethod: models.PaymentNonCash,
	})
	if !errors.Is(err, ErrDiscountTooLarge) {
		t.Errorf("harusnya ErrDiscountTooLarge, dapat: %v", err)
	}
}

func TestCreateOrder_DiskonNegatif(t *testing.T) {
	repo := &mockOrderStore{menuPrices: menuContoh()}
	svc := NewOrderService(repo)

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerName:  "Budi",
		Items:         []models.OrderItem{{MenuItemID: 1, Quantity: 1}},
		Discount:      -1000,
		PaymentMethod: models.PaymentNonCash,
	})
	if !errors.Is(err, ErrDiscountNegative) {
		t.Errorf("harusnya ErrDiscountNegative, dapat: %v", err)
	}
}

func TestCreateOrder_MenuTidakDitemukan(t *testing.T) {
	repo := &mockOrderStore{menuPrices: menuContoh()}
	svc := NewOrderService(repo)

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerName:  "Budi",
		Items:         []models.OrderItem{{MenuItemID: 999, Quantity: 1}},
		PaymentMethod: models.PaymentNonCash,
	})
	if !errors.Is(err, ErrMenuNotExist) {
		t.Errorf("harusnya ErrMenuNotExist, dapat: %v", err)
	}
}

func TestCreateOrder_MetodeBayarNgawur(t *testing.T) {
	repo := &mockOrderStore{menuPrices: menuContoh()}
	svc := NewOrderService(repo)

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerName:  "Budi",
		Items:         []models.OrderItem{{MenuItemID: 1, Quantity: 1}},
		PaymentMethod: "gopay",
	})
	if !errors.Is(err, ErrInvalidPayment) {
		t.Errorf("harusnya ErrInvalidPayment, dapat: %v", err)
	}
}


func TestCreateOrder_DiskonMengurangiTotal(t *testing.T) {
	repo := &mockOrderStore{menuPrices: menuContoh()}
	svc := NewOrderService(repo)

	order, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerName:  "Budi",
		Items:         []models.OrderItem{{MenuItemID: 1, Quantity: 1}},
		Discount:      5000,
		PaymentMethod: models.PaymentCash,
		AmountPaid:    20000,
	})

	if err != nil {
		t.Fatalf("harusnya berhasil, dapat: %v", err)
	}
	if order.Total != 15000 {
		t.Errorf("total salah: mau 15000, dapat %d", order.Total)
	}
	if order.ChangeAmount != 5000 {
		t.Errorf("kembalian salah: mau 5000, dapat %d", order.ChangeAmount)
	}
}

func TestCreateOrder_NamaKosong(t *testing.T) {
	svc := NewOrderService(&mockOrderStore{menuPrices: menuContoh()})

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerName:  "   ",
		Items:         []models.OrderItem{{MenuItemID: 1, Quantity: 1}},
		PaymentMethod: models.PaymentNonCash,
	})
	if !errors.Is(err, ErrEmptyCustomer) {
		t.Errorf("harusnya ErrEmptyCustomer, dapat: %v", err)
	}
}

func TestCreateOrder_ItemKosong(t *testing.T) {
	svc := NewOrderService(&mockOrderStore{})

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerName:  "Budi",
		Items:         []models.OrderItem{},
		PaymentMethod: models.PaymentNonCash,
	})
	if !errors.Is(err, ErrEmptyItems) {
		t.Errorf("harusnya ErrEmptyItems, dapat: %v", err)
	}
}

func TestCreateOrder_QuantityNol(t *testing.T) {
	svc := NewOrderService(&mockOrderStore{menuPrices: menuContoh()})

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerName:  "Budi",
		Items:         []models.OrderItem{{MenuItemID: 1, Quantity: 0}},
		PaymentMethod: models.PaymentNonCash,
	})
	if !errors.Is(err, ErrInvalidQuantity) {
		t.Errorf("harusnya ErrInvalidQuantity, dapat: %v", err)
	}
}

func TestCreateOrder_CountError(t *testing.T) {
	repo := &mockOrderStore{
		countErr:   errors.New("db error"),
		menuPrices: menuContoh(),
	}
	svc := NewOrderService(repo)

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerName:  "Budi",
		Items:         []models.OrderItem{{MenuItemID: 1, Quantity: 1}},
		PaymentMethod: models.PaymentNonCash,
	})
	if err == nil {
		t.Error("harusnya error dari CountOrdersToday diteruskan")
	}
}

// List Orders 

func TestListOrders_FilterValid(t *testing.T) {
	repo := &mockOrderStore{findAllResp: []models.Order{{ID: 1}}}
	svc := NewOrderService(repo)

	orders, err := svc.ListOrders(context.Background(), models.StatusPending)
	if err != nil {
		t.Fatalf("harusnya berhasil, dapat: %v", err)
	}
	if len(orders) != 1 {
		t.Errorf("mau 1 order, dapat %d", len(orders))
	}
}

func TestListOrders_FilterTidakValid(t *testing.T) {
	svc := NewOrderService(&mockOrderStore{})

	_, err := svc.ListOrders(context.Background(), "ngasal")
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("harusnya ErrInvalidStatus, dapat: %v", err)
	}
}

func TestListOrders_TanpaFilter(t *testing.T) {
	repo := &mockOrderStore{findAllResp: []models.Order{{ID: 1}, {ID: 2}}}
	svc := NewOrderService(repo)

	orders, err := svc.ListOrders(context.Background(), "")
	if err != nil {
		t.Fatalf("harusnya berhasil, dapat: %v", err)
	}
	if len(orders) != 2 {
		t.Errorf("mau 2 order, dapat %d", len(orders))
	}
}

// Get Order 

func TestGetOrder_Berhasil(t *testing.T) {
	repo := &mockOrderStore{findByIDResp: &models.Order{ID: 7}}
	svc := NewOrderService(repo)

	order, err := svc.GetOrder(context.Background(), 7)
	if err != nil {
		t.Fatalf("harusnya berhasil, dapat: %v", err)
	}
	if order.ID != 7 {
		t.Errorf("mau id 7, dapat %d", order.ID)
	}
}

// Update Status 

func TestUpdateStatus_PendingKeCooking(t *testing.T) {
	repo := &mockOrderStore{findByIDResp: &models.Order{ID: 1, Status: models.StatusPending}}
	svc := NewOrderService(repo)

	_, err := svc.UpdateStatus(context.Background(), 1, models.StatusCooking)
	if err != nil {
		t.Fatalf("pending→cooking harusnya boleh, dapat: %v", err)
	}
	if repo.lastUpdateStatus != models.StatusCooking {
		t.Errorf("status yang disimpan salah: %s", repo.lastUpdateStatus)
	}
}

func TestUpdateStatus_CookingKeDone(t *testing.T) {
	repo := &mockOrderStore{findByIDResp: &models.Order{ID: 1, Status: models.StatusCooking}}
	svc := NewOrderService(repo)

	_, err := svc.UpdateStatus(context.Background(), 1, models.StatusDone)
	if err != nil {
		t.Fatalf("cooking→done harusnya boleh, dapat: %v", err)
	}
}

func TestUpdateStatus_TransisiIlegal(t *testing.T) {
	repo := &mockOrderStore{findByIDResp: &models.Order{ID: 1, Status: models.StatusPending}}
	svc := NewOrderService(repo)

	_, err := svc.UpdateStatus(context.Background(), 1, models.StatusDone)
	if !errors.Is(err, ErrIllegalTransition) {
		t.Errorf("harusnya ErrIllegalTransition, dapat: %v", err)
	}
}

func TestUpdateStatus_StatusTidakValid(t *testing.T) {
	svc := NewOrderService(&mockOrderStore{})

	_, err := svc.UpdateStatus(context.Background(), 1, "terbang")
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("harusnya ErrInvalidStatus, dapat: %v", err)
	}
}

func TestUpdateStatus_OrderTidakDitemukan(t *testing.T) {
	repo := &mockOrderStore{findByIDErr: errors.New("tidak ada")}
	svc := NewOrderService(repo)

	_, err := svc.UpdateStatus(context.Background(), 999, models.StatusCooking)
	if err == nil {
		t.Error("harusnya error dari FindByID diteruskan")
	}
}

func TestUpdateStatus_DoneTidakBisaBerubah(t *testing.T) {
	repo := &mockOrderStore{findByIDResp: &models.Order{ID: 1, Status: models.StatusDone}}
	svc := NewOrderService(repo)

	_, err := svc.UpdateStatus(context.Background(), 1, models.StatusCooking)
	if !errors.Is(err, ErrIllegalTransition) {
		t.Errorf("done harusnya status akhir, dapat: %v", err)
	}
}

// Cancel Order 

func TestCancelOrder_Berhasil(t *testing.T) {
	repo := &mockOrderStore{findByIDResp: &models.Order{ID: 1, Status: models.StatusPending}}
	svc := NewOrderService(repo)

	_, err := svc.CancelOrder(context.Background(), 1)
	if err != nil {
		t.Fatalf("batal saat pending harusnya boleh, dapat: %v", err)
	}
	if repo.lastUpdateStatus != models.StatusCancelled {
		t.Errorf("status harus cancelled, dapat: %s", repo.lastUpdateStatus)
	}
}

func TestCancelOrder_SudahCooking(t *testing.T) {
	repo := &mockOrderStore{findByIDResp: &models.Order{ID: 1, Status: models.StatusCooking}}
	svc := NewOrderService(repo)

	_, err := svc.CancelOrder(context.Background(), 1)
	if !errors.Is(err, ErrNotEditable) {
		t.Errorf("harusnya ErrNotEditable, dapat: %v", err)
	}
}

func TestCancelOrder_OrderTidakDitemukan(t *testing.T) {
	repo := &mockOrderStore{findByIDErr: errors.New("tidak ada")}
	svc := NewOrderService(repo)

	_, err := svc.CancelOrder(context.Background(), 999)
	if err == nil {
		t.Error("harusnya error dari FindByID diteruskan")
	}
}

// Menu palsu yang dipakai di sebagian besar test.
func menuContoh() map[int64]models.MenuItem {
	return map[int64]models.MenuItem{
		1: {ID: 1, Name: "Nasi Goreng", Price: 20000},
		2: {ID: 2, Name: "Es Teh", Price: 5000},
	}
}