package service

import (
	"context"
	"errors"
	"time"

	"FinalProjectBE/models"
)

var (
	ErrInvalidDate  = errors.New("format tanggal harus YYYY-MM-DD")
	ErrDateInFuture = errors.New("tanggal laporan tidak boleh di masa depan")
)

const defaultTopItemsLimit = 5

type ReportStore interface {
	DailySummary(ctx context.Context, date time.Time) (*models.DailyReport, error)
	TopItems(ctx context.Context, date time.Time, limit int) ([]models.TopItem, error)
}

type ReportService struct {
	repo ReportStore
}

func NewReportService(repo ReportStore) *ReportService {
	return &ReportService{repo: repo}
}

// DailyReport menyusun laporan untuk satu tanggal.
func (s *ReportService) DailyReport(ctx context.Context, dateStr string, limit int) (*models.DailyReport, error) {
	date, err := parseReportDate(dateStr)
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 20 {
		limit = defaultTopItemsLimit
	}

	report, err := s.repo.DailySummary(ctx, date)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.TopItems(ctx, date, limit)
	if err != nil {
		return nil, err
	}
	report.TopItems = items

	return report, nil
}

func parseReportDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, ErrInvalidDate
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if date.After(today) {
		return time.Time{}, ErrDateInFuture
	}

	return date, nil
}