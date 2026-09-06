package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"FinalProjectBE/models"
)

type ReportRepository struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{db: db}
}

// DailySummary menghitung ringkasan order untuk satu tanggal.
func (r *ReportRepository) DailySummary(ctx context.Context, date time.Time) (*models.DailyReport, error) {
	query := `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'cancelled'),
			COALESCE(SUM(subtotal) FILTER (WHERE status <> 'cancelled'), 0),
			COALESCE(SUM(discount) FILTER (WHERE status <> 'cancelled'), 0),
			COALESCE(SUM(total)    FILTER (WHERE status <> 'cancelled'), 0),
			COALESCE(SUM(total)    FILTER (WHERE status <> 'cancelled' AND payment_method = 'cash'), 0),
			COALESCE(SUM(total)    FILTER (WHERE status <> 'cancelled' AND payment_method = 'non_cash'), 0)
		FROM orders
		WHERE created_at::date = $1::date
	`

	var rep models.DailyReport
	err := r.db.QueryRow(ctx, query, date).Scan(
		&rep.TotalOrders,
		&rep.CancelledOrders,
		&rep.GrossRevenue,
		&rep.TotalDiscount,
		&rep.NetRevenue,
		&rep.CashRevenue,
		&rep.NonCashRevenue,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung ringkasan harian: %w", err)
	}

	rep.Date = date.Format("2006-01-02")
	rep.TopItems = []models.TopItem{}
	return &rep, nil
}

// TopItems mengambil menu terlaris pada satu tanggal.
func (r *ReportRepository) TopItems(ctx context.Context, date time.Time, limit int) ([]models.TopItem, error) {
	query := `
		SELECT
			oi.menu_item_id,
			oi.menu_name,
			SUM(oi.quantity)                        AS qty_sold,
			SUM(oi.quantity * oi.price_at_order)    AS revenue
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE o.created_at::date = $1::date
		  AND o.status <> 'cancelled'
		  AND oi.menu_item_id IS NOT NULL
		GROUP BY oi.menu_item_id, oi.menu_name
		ORDER BY qty_sold DESC, revenue DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, date, limit)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil menu terlaris: %w", err)
	}
	defer rows.Close()

	items := make([]models.TopItem, 0)
	for rows.Next() {
		var it models.TopItem
		if err := rows.Scan(&it.MenuItemID, &it.MenuName, &it.QuantitySold, &it.Revenue); err != nil {
			return nil, fmt.Errorf("gagal membaca menu terlaris: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}