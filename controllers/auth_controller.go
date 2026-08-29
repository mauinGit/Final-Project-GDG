package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
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
		"token": result.Token,
		"role":  result.Role,
	})
}