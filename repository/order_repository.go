package repository

import (
	"context"
	"errors"

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

	queryOrder := `INSERT INTO orders (queue_number, customer_name, status)
	               VALUES ($1, $2, $3)
	               RETURNING id, created_at, updated_at`
	err = tx.QueryRow(ctx, queryOrder, order.QueueNumber, order.CustomerName, order.Status).
		Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, err
	}

	queryItem := `INSERT INTO order_items (order_id, menu_name, quantity, note)
	              VALUES ($1, $2, $3, $4)
	              RETURNING id`
	for i := range order.Items {
		order.Items[i].OrderID = order.ID
		err = tx.QueryRow(ctx, queryItem,
			order.ID, order.Items[i].MenuName, order.Items[i].Quantity, order.Items[i].Note).
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

func (r *OrderRepository) FindAll(ctx context.Context, statusFilter string) ([]models.Order, error) {
	queryOrders := `SELECT id, queue_number, customer_name, status, created_at, updated_at
	                FROM orders`
	args := []interface{}{}
	if statusFilter != "" {
		queryOrders += ` WHERE status = $1`
		args = append(args, statusFilter)
	}
	queryOrders += ` ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, queryOrders, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []models.Order{}
	orderIndex := map[int64]int{}
	orderIDs := []int64{}

	for rows.Next() {
		var o models.Order
		o.Items = []models.OrderItem{}
		if err := rows.Scan(&o.ID, &o.QueueNumber, &o.CustomerName,
			&o.Status, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orderIndex[o.ID] = len(orders)
		orders = append(orders, o)
		orderIDs = append(orderIDs, o.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(orderIDs) == 0 {
		return orders, nil
	}

	queryItems := `SELECT id, order_id, menu_name, quantity, COALESCE(note, '')
	               FROM order_items
	               WHERE order_id = ANY($1)`
	itemRows, err := r.db.Query(ctx, queryItems, orderIDs)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var it models.OrderItem
		if err := itemRows.Scan(&it.ID, &it.OrderID, &it.MenuName, &it.Quantity, &it.Note); err != nil {
			return nil, err
		}
		// Tempel item ke order yang sesuai.
		idx := orderIndex[it.OrderID]
		orders[idx].Items = append(orders[idx].Items, it)
	}
	if err := itemRows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *OrderRepository) FindByID(ctx context.Context, id int64) (*models.Order, error) {
	queryOrder := `SELECT id, queue_number, customer_name, status, created_at, updated_at
	               FROM orders WHERE id = $1`

	var o models.Order
	o.Items = []models.OrderItem{}
	err := r.db.QueryRow(ctx, queryOrder, id).Scan(
		&o.ID, &o.QueueNumber, &o.CustomerName, &o.Status, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	queryItems := `SELECT id, order_id, menu_name, quantity, COALESCE(note, '')
	               FROM order_items WHERE order_id = $1`
	rows, err := r.db.Query(ctx, queryItems, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var it models.OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.MenuName, &it.Quantity, &it.Note); err != nil {
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
	          RETURNING id, queue_number, customer_name, status, created_at, updated_at`

	var o models.Order
	o.Items = []models.OrderItem{}
	err := r.db.QueryRow(ctx, query, status, id).Scan(
		&o.ID, &o.QueueNumber, &o.CustomerName, &o.Status, &o.CreatedAt, &o.UpdatedAt,
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