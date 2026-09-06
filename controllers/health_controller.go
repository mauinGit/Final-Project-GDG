package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthController struct {
	pool *pgxpool.Pool
}

func NewHealthController(pool *pgxpool.Pool) *HealthController {
	return &HealthController{pool: pool}
}

// Healthz godoc
// @Summary      Cek aplikasi hidup
// @Description  Menandakan proses aplikasi berjalan. Tidak menyentuh database. Dipakai orchestrator untuk memutuskan perlu restart atau tidak.
// @Tags         health
// @Produce      json
// @Success      200 {object} map[string]interface{} "Aplikasi hidup"
// @Router       /healthz [get]
func (h *HealthController) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

// Readyz godoc
// @Summary      Cek aplikasi siap melayani
// @Description  Memeriksa koneksi database. Mengembalikan 503 jika database tidak terjangkau, menandakan aplikasi belum layak menerima lalu lintas.
// @Tags         health
// @Produce      json
// @Success      200 {object} map[string]interface{} "Siap"
// @Failure      503 {object} map[string]interface{} "Database tidak terjangkau"
// @Router       /readyz [get]
func (h *HealthController) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "not ready",
			"database": "unreachable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ready",
		"database": "ok",
	})
}