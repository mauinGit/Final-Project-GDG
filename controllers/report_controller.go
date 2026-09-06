package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"FinalProjectBE/service"
)

type ReportController struct {
	svc *service.ReportService
}

func NewReportController(svc *service.ReportService) *ReportController {
	return &ReportController{svc: svc}
}

// Daily godoc
// @Summary      Laporan penjualan harian
// @Description  Ringkasan penjualan untuk satu tanggal. Pesanan yang dibatalkan tetap dihitung sebagai aktivitas, tetapi tidak masuk perhitungan omzet. Hanya kasir yang boleh mengakses.
// @Tags         report
// @Produce      json
// @Security     BearerAuth
// @Param        date  query string false "Tanggal laporan, format YYYY-MM-DD. Kosong berarti hari ini"
// @Param        limit query int    false "Jumlah menu terlaris yang ditampilkan" default(5)
// @Success      200 {object} models.DailyReport
// @Failure      403 {object} map[string]string "Bukan kasir"
// @Failure      422 {object} map[string]string "Format tanggal salah atau tanggal di masa depan"
// @Router       /reports/daily [get]
func (ctrl *ReportController) Daily(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))

	report, err := ctrl.svc.DailyReport(c.Request.Context(), c.Query("date"), limit)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidDate),
			errors.Is(err, service.ErrDateInFuture):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyusun laporan"})
		}
		return
	}

	c.JSON(http.StatusOK, report)
}