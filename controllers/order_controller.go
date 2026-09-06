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

func (ctrl *OrderController) List(c *gin.Context) {
	statusFilter := c.Query("status")

	orders, err := ctrl.orderService.ListOrders(c.Request.Context(), statusFilter)
	if err != nil {
		if errors.Is(err, service.ErrInvalidStatus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status filter tidak valid"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan server"})
		return
	}

	c.JSON(http.StatusOK, orders)
}

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