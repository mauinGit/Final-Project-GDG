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

// Create godoc
// @Summary      Tambah menu baru
// @Description  Hanya kasir yang boleh menambah menu. Nama menu unik tanpa peduli huruf besar-kecil.
// @Tags         menu
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body menuRequest true "Data menu"
// @Success      201 {object} models.MenuItem
// @Failure      401 {object} map[string]string "Token tidak valid"
// @Failure      403 {object} map[string]string "Bukan kasir"
// @Failure      409 {object} map[string]string "Nama menu sudah dipakai"
// @Failure      422 {object} map[string]string "Data tidak valid"
// @Router       /menu [post]
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

// List godoc
// @Summary      Daftar menu
// @Description  Menampilkan seluruh menu, bisa disaring per kategori. Kasir dan koki sama-sama boleh mengakses.
// @Tags         menu
// @Produce      json
// @Security     BearerAuth
// @Param        category query string false "Saring berdasarkan kategori"
// @Success      200 {array} models.MenuItem
// @Failure      401 {object} map[string]string "Token tidak valid"
// @Router       /menu [get]
func (ctrl *MenuController) List(c *gin.Context) {
	items, err := ctrl.svc.List(c.Request.Context(), c.Query("category"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mengambil daftar menu"})
		return
	}
	c.JSON(http.StatusOK, items)
}

// GetByID godoc
// @Summary      Detail satu menu
// @Tags         menu
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "ID menu"
// @Success      200 {object} models.MenuItem
// @Failure      400 {object} map[string]string "ID tidak valid"
// @Failure      401 {object} map[string]string "Token tidak valid"
// @Failure      404 {object} map[string]string "Menu tidak ditemukan"
// @Router       /menu/{id} [get]
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

// Update godoc
// @Summary      Ubah data menu
// @Description  Hanya kasir yang boleh mengubah menu.
// @Tags         menu
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "ID menu"
// @Param        request body menuRequest true "Data menu"
// @Success      200 {object} models.MenuItem
// @Failure      400 {object} map[string]string "ID atau input tidak valid"
// @Failure      403 {object} map[string]string "Bukan kasir"
// @Failure      404 {object} map[string]string "Menu tidak ditemukan"
// @Failure      409 {object} map[string]string "Nama menu sudah dipakai"
// @Failure      422 {object} map[string]string "Data tidak valid"
// @Router       /menu/{id} [patch]
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

// Delete godoc
// @Summary      Hapus menu
// @Description  Menu yang sudah pernah dipesan tidak bisa dihapus demi menjaga riwayat transaksi.
// @Tags         menu
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "ID menu"
// @Success      204 "Berhasil dihapus"
// @Failure      400 {object} map[string]string "ID tidak valid"
// @Failure      403 {object} map[string]string "Bukan kasir"
// @Failure      404 {object} map[string]string "Menu tidak ditemukan"
// @Failure      409 {object} map[string]string "Menu sudah pernah dipesan"
// @Router       /menu/{id} [delete]
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