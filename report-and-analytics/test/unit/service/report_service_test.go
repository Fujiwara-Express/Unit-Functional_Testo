package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/report-and-analytics/internal/domain"
	"github.com/report-and-analytics/internal/service"
	"github.com/report-and-analytics/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestService(t *testing.T) (service.ReportService, *mocks.MockReportRepository) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockRepo := mocks.NewMockReportRepository(ctrl)
	svc := service.NewReportService(mockRepo)
	return svc, mockRepo
}

// ── GetOrderReport ────────────────────────────────────────────────────────────

func TestReportService_GetOrderReport_Valid(t *testing.T) {
	svc, mockRepo := newTestService(t)

	expected := &domain.OrderReport{
		Period:       "2026-03",
		TotalOrders:  150000,
		Delivered:    142000,
		Failed:       5000,
		Returned:     3000,
		SuccessRate:  94.67,
		TotalRevenue: 3750000000,
	}
	f := domain.OrderReportFilter{DateFrom: "2026-03-01", DateTo: "2026-03-31", HubID: "HUB_BDG"}
	mockRepo.EXPECT().GetOrderReport(gomock.Any(), f).Return(expected, nil)

	got, err := svc.GetOrderReport(context.Background(), f)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestReportService_GetOrderReport_MissingDateFrom(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetOrderReport(context.Background(), domain.OrderReportFilter{
		DateTo: "2026-03-31",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestReportService_GetOrderReport_MissingDateTo(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetOrderReport(context.Background(), domain.OrderReportFilter{
		DateFrom: "2026-03-01",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestReportService_GetOrderReport_BothDatesMissing(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetOrderReport(context.Background(), domain.OrderReportFilter{})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestReportService_GetOrderReport_RepoError(t *testing.T) {
	svc, mockRepo := newTestService(t)

	f := domain.OrderReportFilter{DateFrom: "2026-03-01", DateTo: "2026-03-31"}
	mockRepo.EXPECT().GetOrderReport(gomock.Any(), f).Return(nil, errors.New("db error"))

	_, err := svc.GetOrderReport(context.Background(), f)
	require.Error(t, err)
}

func TestReportService_GetOrderReport_NoHubID(t *testing.T) {
	svc, mockRepo := newTestService(t)

	f := domain.OrderReportFilter{DateFrom: "2026-03-01", DateTo: "2026-03-31"}
	mockRepo.EXPECT().GetOrderReport(gomock.Any(), f).Return(&domain.OrderReport{}, nil)

	_, err := svc.GetOrderReport(context.Background(), f)
	require.NoError(t, err)
}

// ── GetDeliveryPerformanceReport ─────────────────────────────────────────────

func TestReportService_GetDeliveryPerformanceReport_Valid(t *testing.T) {
	svc, mockRepo := newTestService(t)

	expected := &domain.DeliveryPerformanceReport{
		CourierID:            "CR123",
		Period:               "WEEKLY",
		TotalJobs:            120,
		Delivered:            113,
		Failed:               5,
		Returned:             2,
		SuccessRate:          94.17,
		AvgDeliveryTimeHours: 3.5,
	}
	f := domain.DeliveryPerformanceFilter{CourierID: "CR123", Period: "WEEKLY"}
	mockRepo.EXPECT().GetDeliveryPerformanceReport(gomock.Any(), f).Return(expected, nil)

	got, err := svc.GetDeliveryPerformanceReport(context.Background(), f)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestReportService_GetDeliveryPerformanceReport_MissingCourierID(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetDeliveryPerformanceReport(context.Background(), domain.DeliveryPerformanceFilter{
		Period: "WEEKLY",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestReportService_GetDeliveryPerformanceReport_InvalidPeriod(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetDeliveryPerformanceReport(context.Background(), domain.DeliveryPerformanceFilter{
		CourierID: "CR123",
		Period:    "YEARLY",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestReportService_GetDeliveryPerformanceReport_AllValidPeriods(t *testing.T) {
	periods := []string{"DAILY", "WEEKLY", "MONTHLY"}
	for _, period := range periods {
		t.Run(period, func(t *testing.T) {
			svc, mockRepo := newTestService(t)
			f := domain.DeliveryPerformanceFilter{CourierID: "CR123", Period: period}
			mockRepo.EXPECT().GetDeliveryPerformanceReport(gomock.Any(), f).Return(&domain.DeliveryPerformanceReport{}, nil)

			_, err := svc.GetDeliveryPerformanceReport(context.Background(), f)
			require.NoError(t, err)
		})
	}
}

func TestReportService_GetDeliveryPerformanceReport_RepoError(t *testing.T) {
	svc, mockRepo := newTestService(t)

	f := domain.DeliveryPerformanceFilter{CourierID: "CR123", Period: "WEEKLY"}
	mockRepo.EXPECT().GetDeliveryPerformanceReport(gomock.Any(), f).Return(nil, errors.New("db error"))

	_, err := svc.GetDeliveryPerformanceReport(context.Background(), f)
	require.Error(t, err)
}

// ── GetRevenueReport ──────────────────────────────────────────────────────────

func TestReportService_GetRevenueReport_Valid(t *testing.T) {
	svc, mockRepo := newTestService(t)

	expected := &domain.RevenueReport{
		Period:        "MONTHLY",
		ServiceType:   "REG",
		TotalRevenue:  3750000000,
		TotalOrders:   150000,
		AvgOrderValue: 25000,
	}
	f := domain.RevenueFilter{Period: "MONTHLY", ServiceType: "REG"}
	mockRepo.EXPECT().GetRevenueReport(gomock.Any(), f).Return(expected, nil)

	got, err := svc.GetRevenueReport(context.Background(), f)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestReportService_GetRevenueReport_NoServiceType(t *testing.T) {
	svc, mockRepo := newTestService(t)

	f := domain.RevenueFilter{Period: "MONTHLY"}
	mockRepo.EXPECT().GetRevenueReport(gomock.Any(), f).Return(&domain.RevenueReport{}, nil)

	_, err := svc.GetRevenueReport(context.Background(), f)
	require.NoError(t, err)
}

func TestReportService_GetRevenueReport_InvalidPeriod(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetRevenueReport(context.Background(), domain.RevenueFilter{Period: "QUARTERLY"})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestReportService_GetRevenueReport_EmptyPeriod(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetRevenueReport(context.Background(), domain.RevenueFilter{})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestReportService_GetRevenueReport_RepoError(t *testing.T) {
	svc, mockRepo := newTestService(t)

	f := domain.RevenueFilter{Period: "MONTHLY"}
	mockRepo.EXPECT().GetRevenueReport(gomock.Any(), f).Return(nil, errors.New("db error"))

	_, err := svc.GetRevenueReport(context.Background(), f)
	require.Error(t, err)
}

// ── GetHubPerformanceReport ───────────────────────────────────────────────────

func TestReportService_GetHubPerformanceReport_Valid(t *testing.T) {
	svc, mockRepo := newTestService(t)

	expected := &domain.HubPerformanceReport{
		HubID:                  "HUB_BDG",
		Period:                 "WEEKLY",
		TotalInbound:           8500,
		TotalOutbound:          8200,
		TotalDispatched:        8000,
		AvgDwellTimeHours:      6.2,
		CapacityUtilizationPct: 72.5,
	}
	f := domain.HubPerformanceFilter{HubID: "HUB_BDG", Period: "WEEKLY"}
	mockRepo.EXPECT().GetHubPerformanceReport(gomock.Any(), f).Return(expected, nil)

	got, err := svc.GetHubPerformanceReport(context.Background(), f)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestReportService_GetHubPerformanceReport_MissingHubID(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetHubPerformanceReport(context.Background(), domain.HubPerformanceFilter{
		Period: "WEEKLY",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestReportService_GetHubPerformanceReport_InvalidPeriod(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetHubPerformanceReport(context.Background(), domain.HubPerformanceFilter{
		HubID:  "HUB_BDG",
		Period: "INVALID",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestReportService_GetHubPerformanceReport_RepoError(t *testing.T) {
	svc, mockRepo := newTestService(t)

	f := domain.HubPerformanceFilter{HubID: "HUB_BDG", Period: "WEEKLY"}
	mockRepo.EXPECT().GetHubPerformanceReport(gomock.Any(), f).Return(nil, errors.New("db error"))

	_, err := svc.GetHubPerformanceReport(context.Background(), f)
	require.Error(t, err)
}

func TestReportService_GetHubPerformanceReport_AllValidPeriods(t *testing.T) {
	periods := []string{"DAILY", "WEEKLY", "MONTHLY"}
	for _, period := range periods {
		t.Run(period, func(t *testing.T) {
			svc, mockRepo := newTestService(t)
			f := domain.HubPerformanceFilter{HubID: "HUB_BDG", Period: period}
			mockRepo.EXPECT().GetHubPerformanceReport(gomock.Any(), f).Return(&domain.HubPerformanceReport{}, nil)

			_, err := svc.GetHubPerformanceReport(context.Background(), f)
			require.NoError(t, err)
		})
	}
}
