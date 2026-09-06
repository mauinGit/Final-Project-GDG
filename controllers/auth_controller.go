package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"FinalProjectBE/middleware"
	"FinalProjectBE/service"
)

type AuthController struct {
	authService *service.AuthService
}
func NewAuthController(authService *service.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (ctrl *AuthController) Login(c *gin.Context) {
	var req loginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "input tidak valid: " + err.Error()})
		return
	}

	result, err := ctrl.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "email atau password salah"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan server"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         result.Token,
		"refresh_token": result.RefreshToken,
		"role":          result.Role,
		"expires_in":    result.ExpiresIn,
	})
}

// Refresh godoc
// @Summary      Perbarui access token
// @Description  Menukar refresh token dengan pasangan token baru. Jika refresh token yang sudah pernah dipakai dikirim ulang, seluruh sesi pengguna dicabut.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body refreshRequest true "Refresh token"
// @Success      200 {object} map[string]interface{} "Token baru diterbitkan"
// @Failure      400 {object} map[string]string "Input tidak valid"
// @Failure      401 {object} map[string]string "Token tidak valid, kedaluwarsa, atau sesi dicabut"
// @Router       /auth/refresh [post]
func (ctrl *AuthController) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "input tidak valid: " + err.Error()})
		return
	}

	result, err := ctrl.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRefreshReused):
			// 401 karena sesi memang sudah tidak berlaku,
			// pesannya sengaja menjelaskan alasannya.
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrRefreshExpired),
			errors.Is(err, service.ErrInvalidRefresh):
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan server"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         result.Token,
		"refresh_token": result.RefreshToken,
		"role":          result.Role,
		"expires_in":    result.ExpiresIn,
	})
}

// Logout godoc
// @Summary      Keluar dari sesi
// @Description  Mencabut refresh token sehingga tidak bisa dipakai lagi.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body refreshRequest true "Refresh token yang dicabut"
// @Success      204 "Berhasil keluar"
// @Failure      400 {object} map[string]string "Input tidak valid"
// @Failure      401 {object} map[string]string "Token tidak valid"
// @Router       /auth/logout [post]
func (ctrl *AuthController) Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "input tidak valid: " + err.Error()})
		return
	}

	if err := ctrl.authService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, service.ErrInvalidRefresh) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan server"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Me godoc
// @Summary      Info pengguna saat ini
// @Description  Mengembalikan identitas pemilik access token yang sedang dipakai.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Data pengguna"
// @Failure      401 {object} map[string]string "Token tidak valid"
// @Failure      404 {object} map[string]string "Pengguna tidak ditemukan"
// @Router       /auth/me [get]
func (ctrl *AuthController) Me(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token tidak valid"})
		return
	}

	id, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token tidak valid"})
		return
	}

	user, err := ctrl.authService.Me(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"role":       user.Role,
		"created_at": user.CreatedAt,
	})
}
