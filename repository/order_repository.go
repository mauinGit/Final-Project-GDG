package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"FinalProjectBE/models"
)

var ErrOrderNotFound = errors.New("pesanan tidak ditemukan")

type OrderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
} 

func (r *OrderRepository) Create(ctx context.Context, order *models.Order) (*models.Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	queryOrder := `INSERT INTO orders
	                 (queue_number, customer_name, status, subtotal, discount, total,
	                  payment_method, amount_paid, change_amount)
	               VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	               RETURNING id, created_at, updated_at`
	err = tx.QueryRow(ctx, queryOrder,
		order.QueueNumber, order.CustomerName, order.Status,
		order.Subtotal, order.Discount, order.Total,
		order.PaymentMethod, order.AmountPaid, order.ChangeAmount).
		Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, err
	}

	queryItem := `INSERT INTO order_items
	                (order_id, menu_item_id, menu_name, quantity, note, price_at_order)
	              VALUES ($1, $2, $3, $4, $5, $6)
	              RETURNING id`
	for i := range order.Items {
		order.Items[i].OrderID = order.ID
		err = tx.QueryRow(ctx, queryItem,
			order.ID, order.Items[i].MenuItemID, order.Items[i].MenuName,
			order.Items[i].Quantity, order.Items[i].Note, order.Items[i].PriceAtOrder).
			Scan(&order.Items[i].ID)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return order, nil
}

type OrderFilter struct {
	Status string
	Date   string 
	Page   int
	Limit  int
}

func (r *OrderRepository) FindAll(ctx context.Context, f OrderFilter) ([]models.Order, int, error) {
	where := " WHERE 1=1"
	args := []interface{}{}
	n := 1

	if f.Status != "" {
		where += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, f.Status)
		n++
	}
	if f.Date != "" {
		where += fmt.Sprintf(" AND created_at::date = $%d::date", n)
		args = append(args, f.Date)
		n++
	}

	// Hitung total baris dulu, untuk menyusun meta.
	var totalItems int
	countQuery := `SELECT COUNT(*) FROM orders` + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.Limit

	queryOrders := `SELECT id, queue_number, customer_name, status,
	                       subtotal, discount, total, payment_method, amount_paid, change_amount,
	                       created_at, updated_at
	                FROM orders` + where +
		fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", n, n+1)
	args = append(args, f.Limit, offset)

	rows, err := r.db.Query(ctx, queryOrders, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	orders := []models.Order{}
	orderIndex := map[int64]int{}
	orderIDs := []int64{}

	for rows.Next() {
		var o models.Order
		o.Items = []models.OrderItem{}
		if err := rows.Scan(&o.ID, &o.QueueNumber, &o.CustomerName, &o.Status,
			&o.Subtotal, &o.Discount, &o.Total, &o.PaymentMethod, &o.AmountPaid, &o.ChangeAmount,
			&o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, err
		}
		orderIndex[o.ID] = len(orders)
		orders = append(orders, o)
		orderIDs = append(orderIDs, o.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if len(orderIDs) == 0 {
		return orders, totalItems, nil
	}

	queryItems := `SELECT id, order_id, COALESCE(menu_item_id, 0), menu_name, quantity,
	                      COALESCE(note, ''), price_at_order
	               FROM order_items
	               WHERE order_id = ANY($1)`
	itemRows, err := r.db.Query(ctx, queryItems, orderIDs)
	if err != nil {
		return nil, 0, err
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var it models.OrderItem
		if err := itemRows.Scan(&it.ID, &it.OrderID, &it.MenuItemID, &it.MenuName,
			&it.Quantity, &it.Note, &it.PriceAtOrder); err != nil {
			return nil, 0, err
		}
		idx := orderIndex[it.OrderID]
		orders[idx].Items = append(orders[idx].Items, it)
	}
	if err := itemRows.Err(); err != nil {
		return nil, 0, err
	}

	return orders, totalItems, nil
}

func (r *OrderRepository) FindByID(ctx context.Context, id int64) (*models.Order, error) {
	queryOrder := `SELECT id, queue_number, customer_name, status,
	                  subtotal, discount, total, payment_method, amount_paid, change_amount,
	                  created_at, updated_at
	           FROM orders WHERE id = $1`

	var o models.Order
	o.Items = []models.OrderItem{}
	err := r.db.QueryRow(ctx, queryOrder, id).Scan(
		&o.ID, &o.QueueNumber, &o.CustomerName, &o.Status,
		&o.Subtotal, &o.Discount, &o.Total, &o.PaymentMethod, &o.AmountPaid, &o.ChangeAmount,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	queryItems := `SELECT id, order_id, COALESCE(menu_item_id, 0), menu_name, quantity,
	                      COALESCE(note, ''), price_at_order
	               FROM order_items WHERE order_id = $1`
	rows, err := r.db.Query(ctx, queryItems, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var it models.OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.MenuItemID, &it.MenuName,
			&it.Quantity, &it.Note, &it.PriceAtOrder); err != nil {
			return nil, err
		}
		o.Items = append(o.Items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &o, nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id int64, status string) (*models.Order, error) {
	query := `UPDATE orders
	          SET status = $1, updated_at = now()
	          WHERE id = $2
	          RETURNING id, queue_number, customer_name, status,
	                    subtotal, discount, total, payment_method, amount_paid, change_amount,
	                    created_at, updated_at`

	var o models.Order
	o.Items = []models.OrderItem{}
	err := r.db.QueryRow(ctx, query, status, id).Scan(
		&o.ID, &o.QueueNumber, &o.CustomerName, &o.Status,
		&o.Subtotal, &o.Discount, &o.Total, &o.PaymentMethod, &o.AmountPaid, &o.ChangeAmount,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepository) CountOrdersToday(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM orders WHERE created_at::date = CURRENT_DATE`

	var count int
	if err := r.db.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *OrderRepository) FindMenuPrices(ctx context.Context, ids []int64) (map[int64]models.MenuItem, error) {
	query := `SELECT id, name, price FROM menu_items WHERE id = ANY($1)`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]models.MenuItem)
	for rows.Next() {
		var m models.MenuItem
		if err := rows.Scan(&m.ID, &m.Name, &m.Price); err != nil {
			return nil, err
		}
		result[m.ID] = m
	}
	return result, rows.Err()
}