package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/report-and-analytics/internal/domain"
)

type reportRepository struct {
	db *sql.DB
}

// NewReportRepository creates a ReportRepository backed by *sql.DB (read replica / OLAP).
func NewReportRepository(db *sql.DB) ReportRepository {
	return &reportRepository{db: db}
}

// GetOrderReport aggregates order metrics from fact_orders for the given date range and optional hub.
func (r *reportRepository) GetOrderReport(ctx context.Context, f domain.OrderReportFilter) (*domain.OrderReport, error) {
	query := `
		SELECT
			COUNT(*)                                                        AS total_orders,
			COUNT(*) FILTER (WHERE status = 'DELIVERED')                   AS delivered,
			COUNT(*) FILTER (WHERE status = 'FAILED')                      AS failed,
			COUNT(*) FILTER (WHERE status = 'RETURNED')                    AS returned,
			COALESCE(SUM(revenue), 0)                                      AS total_revenue
		FROM fact_orders
		WHERE order_date BETWEEN $1 AND $2
	`
	args := []interface{}{f.DateFrom, f.DateTo}

	if f.HubID != "" {
		query += " AND hub_id = $3"
		args = append(args, f.HubID)
	}

	row := r.db.QueryRowContext(ctx, query, args...)

	var rep domain.OrderReport
	if err := row.Scan(
		&rep.TotalOrders,
		&rep.Delivered,
		&rep.Failed,
		&rep.Returned,
		&rep.TotalRevenue,
	); err != nil {
		return nil, fmt.Errorf("GetOrderReport: %w", err)
	}

	rep.Period = periodFromDates(f.DateFrom, f.DateTo)
	if rep.TotalOrders > 0 {
		rep.SuccessRate = round2(float64(rep.Delivered) / float64(rep.TotalOrders) * 100)
	}
	return &rep, nil
}

// GetDeliveryPerformanceReport aggregates delivery metrics per courier from fact_deliveries.
func (r *reportRepository) GetDeliveryPerformanceReport(ctx context.Context, f domain.DeliveryPerformanceFilter) (*domain.DeliveryPerformanceReport, error) {
	query := `
		SELECT
			COUNT(*)                                                        AS total_jobs,
			COUNT(*) FILTER (WHERE status = 'DELIVERED')                   AS delivered,
			COUNT(*) FILTER (WHERE status = 'FAILED')                      AS failed,
			COUNT(*) FILTER (WHERE status = 'RETURNED')                    AS returned,
			COALESCE(AVG(delivery_time_hours), 0)                          AS avg_delivery_time_hours
		FROM fact_deliveries
		WHERE courier_id = $1
		  AND period_type = $2
	`
	row := r.db.QueryRowContext(ctx, query, f.CourierID, f.Period)

	var rep domain.DeliveryPerformanceReport
	if err := row.Scan(
		&rep.TotalJobs,
		&rep.Delivered,
		&rep.Failed,
		&rep.Returned,
		&rep.AvgDeliveryTimeHours,
	); err != nil {
		return nil, fmt.Errorf("GetDeliveryPerformanceReport: %w", err)
	}

	rep.CourierID = f.CourierID
	rep.Period = f.Period
	rep.AvgDeliveryTimeHours = round2(rep.AvgDeliveryTimeHours)
	if rep.TotalJobs > 0 {
		rep.SuccessRate = round2(float64(rep.Delivered) / float64(rep.TotalJobs) * 100)
	}
	return &rep, nil
}

// GetRevenueReport aggregates revenue metrics from fact_orders for the given period and optional service type.
func (r *reportRepository) GetRevenueReport(ctx context.Context, f domain.RevenueFilter) (*domain.RevenueReport, error) {
	query := `
		SELECT
			COALESCE(SUM(revenue), 0)  AS total_revenue,
			COUNT(*)                   AS total_orders
		FROM fact_orders
		WHERE period_type = $1
	`
	args := []interface{}{f.Period}

	if f.ServiceType != "" {
		query += " AND service_type = $2"
		args = append(args, f.ServiceType)
	}

	row := r.db.QueryRowContext(ctx, query, args...)

	var rep domain.RevenueReport
	if err := row.Scan(&rep.TotalRevenue, &rep.TotalOrders); err != nil {
		return nil, fmt.Errorf("GetRevenueReport: %w", err)
	}

	rep.Period = f.Period
	rep.ServiceType = f.ServiceType
	if rep.TotalOrders > 0 {
		rep.AvgOrderValue = round2(float64(rep.TotalRevenue) / float64(rep.TotalOrders))
	}
	return &rep, nil
}

// GetHubPerformanceReport aggregates hub throughput and utilisation metrics.
func (r *reportRepository) GetHubPerformanceReport(ctx context.Context, f domain.HubPerformanceFilter) (*domain.HubPerformanceReport, error) {
	query := `
		SELECT
			COALESCE(SUM(inbound_count), 0)          AS total_inbound,
			COALESCE(SUM(outbound_count), 0)         AS total_outbound,
			COALESCE(SUM(dispatched_count), 0)       AS total_dispatched,
			COALESCE(AVG(avg_dwell_time_hours), 0)   AS avg_dwell_time_hours,
			COALESCE(AVG(capacity_utilization_pct), 0) AS capacity_utilization_pct
		FROM fact_hub_performance
		WHERE hub_id = $1
		  AND period_type = $2
	`
	row := r.db.QueryRowContext(ctx, query, f.HubID, f.Period)

	var rep domain.HubPerformanceReport
	if err := row.Scan(
		&rep.TotalInbound,
		&rep.TotalOutbound,
		&rep.TotalDispatched,
		&rep.AvgDwellTimeHours,
		&rep.CapacityUtilizationPct,
	); err != nil {
		return nil, fmt.Errorf("GetHubPerformanceReport: %w", err)
	}

	rep.HubID = f.HubID
	rep.Period = f.Period
	rep.AvgDwellTimeHours = round2(rep.AvgDwellTimeHours)
	rep.CapacityUtilizationPct = round2(rep.CapacityUtilizationPct)
	return &rep, nil
}

// periodFromDates derives a human-readable period label from a date range.
func periodFromDates(from, to string) string {
	if len(from) >= 7 {
		return from[:7] // "YYYY-MM"
	}
	return from
}

// round2 rounds a float64 to 2 decimal places.
func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
