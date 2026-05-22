package service

import (
	"context"
	"fmt"

	"github.com/report-and-analytics/internal/domain"
	"github.com/report-and-analytics/internal/repository"
)

type reportService struct {
	repo repository.ReportRepository
}

// NewReportService creates a new ReportService.
func NewReportService(repo repository.ReportRepository) ReportService {
	return &reportService{repo: repo}
}

func (s *reportService) GetOrderReport(ctx context.Context, f domain.OrderReportFilter) (*domain.OrderReport, error) {
	if f.DateFrom == "" || f.DateTo == "" {
		return nil, fmt.Errorf("%w: date_from and date_to are required", domain.ErrValidation)
	}
	return s.repo.GetOrderReport(ctx, f)
}

func (s *reportService) GetDeliveryPerformanceReport(ctx context.Context, f domain.DeliveryPerformanceFilter) (*domain.DeliveryPerformanceReport, error) {
	if f.CourierID == "" {
		return nil, fmt.Errorf("%w: courier_id is required", domain.ErrValidation)
	}
	if err := validatePeriod(f.Period); err != nil {
		return nil, err
	}
	return s.repo.GetDeliveryPerformanceReport(ctx, f)
}

func (s *reportService) GetRevenueReport(ctx context.Context, f domain.RevenueFilter) (*domain.RevenueReport, error) {
	if err := validatePeriod(f.Period); err != nil {
		return nil, err
	}
	return s.repo.GetRevenueReport(ctx, f)
}

func (s *reportService) GetHubPerformanceReport(ctx context.Context, f domain.HubPerformanceFilter) (*domain.HubPerformanceReport, error) {
	if f.HubID == "" {
		return nil, fmt.Errorf("%w: hub_id is required", domain.ErrValidation)
	}
	if err := validatePeriod(f.Period); err != nil {
		return nil, err
	}
	return s.repo.GetHubPerformanceReport(ctx, f)
}

// validatePeriod checks that the period value is one of the accepted values.
func validatePeriod(period string) error {
	switch period {
	case "DAILY", "WEEKLY", "MONTHLY":
		return nil
	default:
		return fmt.Errorf("%w: period must be DAILY, WEEKLY, or MONTHLY", domain.ErrValidation)
	}
}
