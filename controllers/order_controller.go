package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"FinalProjectBE/models"
	"FinalProjectBE/repository"
	"FinalProjectBE/service"
	"FinalProjectBE/ws"
)

type OrderController struct {
	orderService *service.OrderService
	hub          *ws.Hub
}

func NewOrderController(orderService *service.OrderService, hub *ws.Hub) *OrderController {
	return &OrderController{orderService: orderService, hub: hub}
}

type orderItemRequest struct {
	MenuItemID int64  `json:"menu_item_id" binding:"required,gt=0"`
	Quantity   int    `json:"quantity" binding:"required,gt=0"`
	Note       string `json:"note"`
}

type createOrderRequest struct {
	CustomerName  string             `json:"customer_name" binding:"required"`
	Items         []orderItemRequest `json:"items" binding:"required,min=1,dive"`
	Discount      int                `json:"discount"`
	PaymentMethod string             `json:"payment_method" binding:"required"`
	AmountPaid    int                `json:"amount_paid"`
}

type updateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// Create godoc
// @Summary      Buat pesanan baru
// @Description  Kasir membuat pesanan. Nama dan harga item diambil otomatis dari data menu, bukan dari input. Nomor antrean dihitung otomatis dan direset setiap hari.
// @Tags         order
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body createOrderRequest true "Data pesanan"
// @Success      201 {object} models.Order
// @Failure      400 {object} map[string]string "Input tidak valid"
// @Failure      403 {object} map[string]string "Bukan kasir"
// @Failure      422 {object} map[string]string "Menu tidak ada, diskon melebihi subtotal, atau uang kurang"
// @Router       /orders [post]
func (ctrl *OrderController) Create(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "input tidak valid: " + err.Error()})
		return
	}

	items := make([]models.OrderItem, len(req.Items))
	for i, it := range req.Items {
		items[i] = models.OrderItem{
			MenuItemID: it.MenuItemID,
			Quantity:   it.Quantity,
			Note:       it.Note,
		}
	}

	order, err := ctrl.orderService.CreateOrder(c.Request.Context(), service.CreateOrderInput{
		CustomerName:  req.CustomerName,
		Items:         items,
		Discount:      req.Discount,
		PaymentMethod: req.PaymentMethod,
		AmountPaid:    req.AmountPaid,
	})
	if err != nil {
		writeCreateOrderError(c, err)
		return
	}

	ctrl.hub.Broadcast("order_created", order)

	c.JSON(http.StatusCreated, order)
}

// List godoc
// @Summary      Daftar pesanan
// @Description  Menampilkan pesanan dengan halaman. Bisa disaring per status dan per tanggal. Terurut dari yang terbaru.
// @Tags         order
// @Produce      json
// @Security     BearerAuth
// @Param        status query string false "Saring status" Enums(pending, cooking, done, cancelled)
// @Param        date   query string false "Saring tanggal, format YYYY-MM-DD"
// @Param        page   query int    false "Halaman, mulai dari 1" default(1)
// @Param        limit  query int    false "Baris per halaman, maksimal 100" default(10)
// @Success      200 {object} models.OrderListResult
// @Failure      401 {object} map[string]string "Token tidak valid"
// @Failure      422 {object} map[string]string "Status atau tanggal tidak valid"
// @Router       /orders [get]
func (ctrl *OrderController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	result, err := ctrl.orderService.ListOrders(c.Request.Context(), repository.OrderFilter{
		Status: c.Query("status"),
		Date:   c.Query("date"),
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidStatus):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "status filter tidak valid"})
		case errors.Is(err, service.ErrInvalidDateFilter):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan server"})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetByID godoc
// @Summary      Detail satu pesanan
// @Tags         order
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "ID pesanan"
// @Success      200 {object} models.Order
// @Failure      400 {object} map[string]string "ID tidak valid"
// @Failure      404 {object} map[string]string "Pesanan tidak ditemukan"
// @Router       /orders/{id} [get]
func (ctrl *OrderController) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}

	order, err := ctrl.orderService.GetOrder(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "pesanan tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan server"})
		return
	}

	c.JSON(http.StatusOK, order)
}

// UpdateStatus godoc
// @Summary      Ubah status pesanan
// @Description  Hanya koki yang boleh mengubah status. Alur yang diizinkan: pending ke cooking, lalu cooking ke done. Perubahan di luar alur itu ditolak.
// @Tags         order
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "ID pesanan"
// @Param        request body updateStatusRequest true "Status baru"
// @Success      200 {object} models.Order
// @Failure      400 {object} map[string]string "ID atau status tidak dikenal"
// @Failure      403 {object} map[string]string "Bukan koki"
// @Failure      404 {object} map[string]string "Pesanan tidak ditemukan"
// @Failure      422 {object} map[string]string "Perubahan status tidak diperbolehkan"
// @Router       /orders/{id}/status [patch]
func (ctrl *OrderController) UpdateStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}

	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "input tidak valid: " + err.Error()})
		return
	}

	order, err := ctrl.orderService.UpdateStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrOrderNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "pesanan tidak ditemukan"})
		case errors.Is(err, service.ErrIllegalTransition):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "perubahan status tidak diperbolehkan"})
		case errors.Is(err, service.ErrInvalidStatus):
			c.JSON(http.StatusBadRequest, gin.H{"error": "status tidak dikenal"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan server"})
		}
		return
	}

	ctrl.hub.Broadcast("order_updated", order)

	c.JSON(http.StatusOK, order)
}

// Cancel godoc
// @Summary      Batalkan pesanan
// @Description  Hanya kasir yang boleh membatalkan, dan hanya selama pesanan masih berstatus pending.
// @Tags         order
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "ID pesanan"
// @Success      200 {object} models.Order
// @Failure      400 {object} map[string]string "ID tidak valid"
// @Failure      403 {object} map[string]string "Bukan kasir"
// @Failure      404 {object} map[string]string "Pesanan tidak ditemukan"
// @Failure      409 {object} map[string]string "Pesanan sudah tidak bisa dibatalkan"
// @Router       /orders/{id} [delete]
func (ctrl *OrderController) Cancel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}

	order, err := ctrl.orderService.CancelOrder(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrOrderNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "pesanan tidak ditemukan"})
		case errors.Is(err, service.ErrNotEditable):
			c.JSON(http.StatusConflict, gin.H{"error": "pesanan tidak bisa dibatalkan (bukan pending)"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan server"})
		}
		return
	}

	ctrl.hub.Broadcast("order_updated", order)

	c.JSON(http.StatusOK, order)
}

func writeCreateOrderError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrMenuNotExist):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrInvalidPayment),
		errors.Is(err, service.ErrDiscountNegative),
		errors.Is(err, service.ErrDiscountTooLarge),
		errors.Is(err, service.ErrInsufficientPaid),
		errors.Is(err, service.ErrEmptyCustomer),
		errors.Is(err, service.ErrEmptyItems),
		errors.Is(err, service.ErrInvalidQuantity):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})

	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan server"})
	}
}