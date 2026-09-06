package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"FinalProjectBE/models"
	"FinalProjectBE/repository"
	"FinalProjectBE/service"
)

type MenuController struct {
	svc *service.MenuService
}

func NewMenuController(svc *service.MenuService) *MenuController {
	return &MenuController{svc: svc}
}

type menuRequest struct {
	Name     string  `json:"name" binding:"required"`
	Price    int     `json:"price"`
	Category string  `json:"category" binding:"required"`
	ImageURL *string `json:"image_url"`
}

func (ctrl *MenuController) Create(c *gin.Context) {
	var req menuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item := &models.MenuItem{
		Name:     req.Name,
		Price:    req.Price,
		Category: req.Category,
		ImageURL: req.ImageURL,
	}

	if err := ctrl.svc.Create(c.Request.Context(), item); err != nil {
		writeMenuError(c, err)
		return
	}

	c.JSON(http.StatusCreated, item)
}

func (ctrl *MenuController) List(c *gin.Context) {
	items, err := ctrl.svc.List(c.Request.Context(), c.Query("category"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mengambil daftar menu"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (ctrl *MenuController) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}

	item, err := ctrl.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		writeMenuError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (ctrl *MenuController) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}

	var req menuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item := &models.MenuItem{
		ID:       id,
		Name:     req.Name,
		Price:    req.Price,
		Category: req.Category,
		ImageURL: req.ImageURL,
	}

	if err := ctrl.svc.Update(c.Request.Context(), item); err != nil {
		writeMenuError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (ctrl *MenuController) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}

	if err := ctrl.svc.Delete(c.Request.Context(), id); err != nil {
		writeMenuError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// writeMenuError menerjemahkan error domain menjadi status HTTP yang sesuai.
func writeMenuError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrMenuItemNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrMenuNameTaken):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrMenuInUse):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrMenuNameRequired),
		errors.Is(err, service.ErrMenuCategoryEmpty),
		errors.Is(err, service.ErrMenuPriceInvalid):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})

	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan internal"})
	}
}