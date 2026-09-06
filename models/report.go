package models

type TopItem struct {
	MenuItemID   int64  `json:"menu_item_id"`
	MenuName     string `json:"menu_name"`
	QuantitySold int    `json:"quantity_sold"`
	Revenue      int    `json:"revenue"`
}

type DailyReport struct {
	Date            string    `json:"date"`
	TotalOrders     int       `json:"total_orders"`
	CancelledOrders int       `json:"cancelled_orders"`
	GrossRevenue    int       `json:"gross_revenue"`
	TotalDiscount   int       `json:"total_discount"`
	NetRevenue      int       `json:"net_revenue"`
	CashRevenue     int       `json:"cash_revenue"`
	NonCashRevenue  int       `json:"non_cash_revenue"`
	TopItems        []TopItem `json:"top_items"`
}