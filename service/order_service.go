package service

import (
	"context"
	"errors"

	"FinalProjectBE/models"
)

// Error-error logika bisnis pesanan.
var (
	ErrEmptyItems       = errors.New("pesanan harus memiliki minimal satu item")
	ErrEmptyCustomer    = errors.New("nama pelanggan wajib diisi")
	ErrInvalidQuantity  = errors.New("jumlah item harus lebih dari nol")
	ErrInvalidStatus    = errors.New("status tidak dikenal")
	ErrIllegalTransition = errors.New("perubahan status tidak diperbolehkan")
	ErrNotEditable      = errors.New("pesanan hanya bisa diubah saat masih pending")
)

// OrderStore adalah kontrak yang dibutuhkan OrderService dari repository.
// Dengan interface ini, service bisa diuji memakai mock tanpa database.
type OrderStore interface {
	Create(ctx context.Context, order *models.Order) (*models.Order, error)
	FindAll(ctx context.Context, statusFilter string) ([]models.Order, error)
	FindByID(ctx context.Context, id int64) (*models.Order, error)
	UpdateStatus(ctx context.Context, id int64, status string) (*models.Order, error)
	CountOrdersToday(ctx context.Context) (int, error)
}

// OrderService menampung logika bisnis pesanan.
type OrderService struct {
	repo OrderStore
}

// NewOrderService membuat instance OrderService.
func NewOrderService(repo OrderStore) *OrderService {
	return &OrderService{repo: repo}
}

// CreateOrder memvalidasi input, menentukan nomor urut harian, lalu menyimpan pesanan.
func (s *OrderService) CreateOrder(ctx context.Context, customerName string, items []models.OrderItem) (*models.Order, error) {
	if customerName == "" {
		return nil, ErrEmptyCustomer
	}
	if len(items) == 0 {
		return nil, ErrEmptyItems
	}
	for _, it := range items {
		if it.MenuName == "" {
			return nil, ErrEmptyItems
		}
		if it.Quantity <= 0 {
			return nil, ErrInvalidQuantity
		}
	}

	// Nomor urut harian: jumlah pesanan hari ini + 1.
	count, err := s.repo.CountOrdersToday(ctx)
	if err != nil {
		return nil, err
	}

	order := &models.Order{
		QueueNumber:  count + 1,
		CustomerName: customerName,
		Status:       models.StatusPending,
		Items:        items,
	}

	return s.repo.Create(ctx, order)
}

// ListOrders mengembalikan daftar pesanan, opsional difilter status.
func (s *OrderService) ListOrders(ctx context.Context, statusFilter string) ([]models.Order, error) {
	if statusFilter != "" && !isValidStatus(statusFilter) {
		return nil, ErrInvalidStatus
	}
	return s.repo.FindAll(ctx, statusFilter)
}

// GetOrder mengembalikan satu pesanan berdasarkan id.
func (s *OrderService) GetOrder(ctx context.Context, id int64) (*models.Order, error) {
	return s.repo.FindByID(ctx, id)
}

// UpdateStatus mengubah status pesanan dengan menegakkan aturan transisi.
func (s *OrderService) UpdateStatus(ctx context.Context, id int64, newStatus string) (*models.Order, error) {
	if !isValidStatus(newStatus) {
		return nil, ErrInvalidStatus
	}

	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !isAllowedTransition(current.Status, newStatus) {
		return nil, ErrIllegalTransition
	}

	return s.repo.UpdateStatus(ctx, id, newStatus)
}

// CancelOrder membatalkan pesanan, hanya boleh saat masih pending.
func (s *OrderService) CancelOrder(ctx context.Context, id int64) (*models.Order, error) {
	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Status != models.StatusPending {
		return nil, ErrNotEditable
	}
	return s.repo.UpdateStatus(ctx, id, models.StatusCancelled)
}

// isValidStatus memastikan status termasuk salah satu nilai yang dikenal.
func isValidStatus(status string) bool {
	switch status {
	case models.StatusPending, models.StatusCooking, models.StatusDone, models.StatusCancelled:
		return true
	default:
		return false
	}
}

// isAllowedTransition menegakkan alur: pending→cooking→done, dan pending→cancelled.
func isAllowedTransition(from, to string) bool {
	switch from {
	case models.StatusPending:
		return to == models.StatusCooking || to == models.StatusCancelled
	case models.StatusCooking:
		return to == models.StatusDone
	default:
		// done & cancelled adalah status akhir, tidak bisa berubah lagi.
		return false
	}
}