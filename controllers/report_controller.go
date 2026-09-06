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