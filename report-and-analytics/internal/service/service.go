package service

import (
	"context"

	"github.com/report-and-analytics/internal/domain"
)

// ReportService defines the business logic interface for report generation.
type ReportService interface {
	GetOrderReport(ctx context.Context, f domain.OrderReportFilter) (*domain.OrderReport, error)
	GetDeliveryPerformanceReport(ctx context.Context, f domain.DeliveryPerformanceFilter) (*domain.DeliveryPerformanceReport, error)
	GetRevenueReport(ctx context.Context, f domain.RevenueFilter) (*domain.RevenueReport, error)
	GetHubPerformanceReport(ctx context.Context, f domain.HubPerformanceFilter) (*domain.HubPerformanceReport, error)
}
