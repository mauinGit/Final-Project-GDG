package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"FinalProjectBE/models"
)

var (
	ErrEmptyItems        = errors.New("pesanan harus memiliki minimal satu item")
	ErrEmptyCustomer     = errors.New("nama pelanggan wajib diisi")
	ErrInvalidQuantity   = errors.New("jumlah item harus lebih dari nol")
	ErrInvalidStatus     = errors.New("status tidak dikenal")
	ErrIllegalTransition = errors.New("perubahan status tidak diperbolehkan")
	ErrNotEditable       = errors.New("pesanan hanya bisa diubah saat masih pending")

	ErrMenuNotExist       = errors.New("menu yang dipesan tidak ditemukan")
	ErrInvalidPayment     = errors.New("metode pembayaran harus 'cash' atau 'non_cash'")
	ErrDiscountNegative   = errors.New("diskon tidak boleh negatif")
	ErrDiscountTooLarge   = errors.New("diskon tidak boleh melebihi subtotal")
	ErrInsufficientPaid   = errors.New("uang yang diterima kurang dari total tagihan")
)

type OrderStore interface {
	Create(ctx context.Context, order *models.Order) (*models.Order, error)
	FindAll(ctx context.Context, statusFilter string) ([]models.Order, error)
	FindByID(ctx context.Context, id int64) (*models.Order, error)
	UpdateStatus(ctx context.Context, id int64, status string) (*models.Order, error)
	CountOrdersToday(ctx context.Context) (int, error)
	FindMenuPrices(ctx context.Context, ids []int64) (map[int64]models.MenuItem, error)
}

type OrderService struct {
	repo OrderStore
}

func NewOrderService(repo OrderStore) *OrderService {
	return &OrderService{repo: repo}
}

// CreateOrderInput menampung semua yang dikirim kasir saat membuat pesanan.
type CreateOrderInput struct {
	CustomerName  string
	Items         []models.OrderItem
	Discount      int
	PaymentMethod string
	AmountPaid    int
}

func (s *OrderService) CreateOrder(ctx context.Context, in CreateOrderInput) (*models.Order, error) {
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	if in.CustomerName == "" {
		return nil, ErrEmptyCustomer
	}
	if len(in.Items) == 0 {
		return nil, ErrEmptyItems
	}
	if in.PaymentMethod != models.PaymentCash && in.PaymentMethod != models.PaymentNonCash {
		return nil, ErrInvalidPayment
	}
	if in.Discount < 0 {
		return nil, ErrDiscountNegative
	}

	// Kumpulkan ID menu yang dipesan.
	ids := make([]int64, 0, len(in.Items))
	for _, it := range in.Items {
		if it.Quantity <= 0 {
			return nil, ErrInvalidQuantity
		}
		if it.MenuItemID <= 0 {
			return nil, ErrMenuNotExist
		}
		ids = append(ids, it.MenuItemID)
	}

	// Ambil harga resmi dari tabel menu — bukan dari input kasir.
	menus, err := s.repo.FindMenuPrices(ctx, ids)
	if err != nil {
		return nil, err
	}

	subtotal := 0
	items := make([]models.OrderItem, len(in.Items))
	for i, it := range in.Items {
		menu, ok := menus[it.MenuItemID]
		if !ok {
			return nil, fmt.Errorf("%w (id: %d)", ErrMenuNotExist, it.MenuItemID)
		}

		items[i] = models.OrderItem{
			MenuItemID:   menu.ID,
			MenuName:     menu.Name,  // snapshot nama
			PriceAtOrder: menu.Price, // snapshot harga
			Quantity:     it.Quantity,
			Note:         it.Note,
		}
		subtotal += menu.Price * it.Quantity
	}

	if in.Discount > subtotal {
		return nil, ErrDiscountTooLarge
	}
	total := subtotal - in.Discount

	amountPaid, change := 0, 0
	if in.PaymentMethod == models.PaymentCash {
		if in.AmountPaid < total {
			return nil, ErrInsufficientPaid
		}
		amountPaid = in.AmountPaid
		change = in.AmountPaid - total
	}

	count, err := s.repo.CountOrdersToday(ctx)
	if err != nil {
		return nil, err
	}

	order := &models.Order{
		QueueNumber:   count + 1,
		CustomerName:  in.CustomerName,
		Status:        models.StatusPending,
		Items:         items,
		Subtotal:      subtotal,
		Discount:      in.Discount,
		Total:         total,
		PaymentMethod: in.PaymentMethod,
		AmountPaid:    amountPaid,
		ChangeAmount:  change,
	}

	return s.repo.Create(ctx, order)
}

func (s *OrderService) ListOrders(ctx context.Context, statusFilter string) ([]models.Order, error) {
	if statusFilter != "" && !isValidStatus(statusFilter) {
		return nil, ErrInvalidStatus
	}
	return s.repo.FindAll(ctx, statusFilter)
}

func (s *OrderService) GetOrder(ctx context.Context, id int64) (*models.Order, error) {
	return s.repo.FindByID(ctx, id)
}

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

func isValidStatus(status string) bool {
	switch status {
	case models.StatusPending, models.StatusCooking, models.StatusDone, models.StatusCancelled:
		return true
	default:
		return false
	}
}

func isAllowedTransition(from, to string) bool {
	switch from {
	case models.StatusPending:
		return to == models.StatusCooking || to == models.StatusCancelled
	case models.StatusCooking:
		return to == models.StatusDone
	default:
		return false
	}
}