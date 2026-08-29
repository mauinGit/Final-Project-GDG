package service

import (
	"context"
	"errors"
	"testing"

	"FinalProjectBE/models"
)

// mockOrderStore adalah repository palsu untuk menguji OrderService tanpa database.
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

	// merekam argumen yang dipakai, untuk verifikasi.
	lastUpdateStatus string
}

func (m *mockOrderStore) Create(ctx context.Context, order *models.Order) (*models.Order, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	order.ID = 99 // simulasikan id hasil insert
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

// ---------- CreateOrder ----------

func TestCreateOrder_Berhasil(t *testing.T) {
	repo := &mockOrderStore{countToday: 4}
	svc := NewOrderService(repo)

	items := []models.OrderItem{{MenuName: "Nasi Goreng", Quantity: 2}}
	order, err := svc.CreateOrder(context.Background(), "Budi", items)

	if err != nil {
		t.Fatalf("harusnya berhasil, dapat error: %v", err)
	}
	if order.QueueNumber != 5 {
		t.Errorf("nomor urut salah: mau 5 (4+1), dapat %d", order.QueueNumber)
	}
	if order.Status != models.StatusPending {
		t.Errorf("status awal harus pending, dapat %s", order.Status)
	}
}

func TestCreateOrder_NamaKosong(t *testing.T) {
	svc := NewOrderService(&mockOrderStore{})
	items := []models.OrderItem{{MenuName: "Teh", Quantity: 1}}

	_, err := svc.CreateOrder(context.Background(), "", items)
	if !errors.Is(err, ErrEmptyCustomer) {
		t.Errorf("harusnya ErrEmptyCustomer, dapat: %v", err)
	}
}

func TestCreateOrder_ItemKosong(t *testing.T) {
	svc := NewOrderService(&mockOrderStore{})

	_, err := svc.CreateOrder(context.Background(), "Budi", []models.OrderItem{})
	if !errors.Is(err, ErrEmptyItems) {
		t.Errorf("harusnya ErrEmptyItems, dapat: %v", err)
	}
}

func TestCreateOrder_MenuNameKosong(t *testing.T) {
	svc := NewOrderService(&mockOrderStore{})
	items := []models.OrderItem{{MenuName: "", Quantity: 1}}

	_, err := svc.CreateOrder(context.Background(), "Budi", items)
	if !errors.Is(err, ErrEmptyItems) {
		t.Errorf("harusnya ErrEmptyItems, dapat: %v", err)
	}
}

func TestCreateOrder_QuantityNol(t *testing.T) {
	svc := NewOrderService(&mockOrderStore{})
	items := []models.OrderItem{{MenuName: "Teh", Quantity: 0}}

	_, err := svc.CreateOrder(context.Background(), "Budi", items)
	if !errors.Is(err, ErrInvalidQuantity) {
		t.Errorf("harusnya ErrInvalidQuantity, dapat: %v", err)
	}
}

func TestCreateOrder_CountError(t *testing.T) {
	repo := &mockOrderStore{countErr: errors.New("db error")}
	svc := NewOrderService(repo)
	items := []models.OrderItem{{MenuName: "Teh", Quantity: 1}}

	_, err := svc.CreateOrder(context.Background(), "Budi", items)
	if err == nil {
		t.Error("harusnya error dari CountOrdersToday diteruskan")
	}
}

// ---------- ListOrders ----------

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

// ---------- GetOrder ----------

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

// ---------- UpdateStatus ----------

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

	// pending langsung ke done tidak diperbolehkan.
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

// ---------- CancelOrder ----------

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