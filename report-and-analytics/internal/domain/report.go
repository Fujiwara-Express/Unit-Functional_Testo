package domain

// OrderReport holds aggregated order statistics for a given period and optional hub.
type OrderReport struct {
	Period       string  `json:"period"`
	TotalOrders  int64   `json:"total_orders"`
	Delivered    int64   `json:"delivered"`
	Failed       int64   `json:"failed"`
	Returned     int64   `json:"returned"`
	SuccessRate  float64 `json:"success_rate"`
	TotalRevenue int64   `json:"total_revenue"`
}

// DeliveryPerformanceReport holds per-courier delivery performance for a period.
type DeliveryPerformanceReport struct {
	CourierID           string  `json:"courier_id"`
	Period              string  `json:"period"`
	TotalJobs           int64   `json:"total_jobs"`
	Delivered           int64   `json:"delivered"`
	Failed              int64   `json:"failed"`
	Returned            int64   `json:"returned"`
	SuccessRate         float64 `json:"success_rate"`
	AvgDeliveryTimeHours float64 `json:"avg_delivery_time_hours"`
}

// RevenueReport holds revenue aggregates for a period and optional service type.
type RevenueReport struct {
	Period        string  `json:"period"`
	ServiceType   string  `json:"service_type"`
	TotalRevenue  int64   `json:"total_revenue"`
	TotalOrders   int64   `json:"total_orders"`
	AvgOrderValue float64 `json:"avg_order_value"`
}

// HubPerformanceReport holds throughput and utilisation metrics for a hub.
type HubPerformanceReport struct {
	HubID                  string  `json:"hub_id"`
	Period                 string  `json:"period"`
	TotalInbound           int64   `json:"total_inbound"`
	TotalOutbound          int64   `json:"total_outbound"`
	TotalDispatched        int64   `json:"total_dispatched"`
	AvgDwellTimeHours      float64 `json:"avg_dwell_time_hours"`
	CapacityUtilizationPct float64 `json:"capacity_utilization_pct"`
}

// OrderReportFilter holds query parameters for the order report.
type OrderReportFilter struct {
	DateFrom string // YYYY-MM-DD
	DateTo   string // YYYY-MM-DD
	HubID    string // optional
}

// DeliveryPerformanceFilter holds query parameters for the delivery performance report.
type DeliveryPerformanceFilter struct {
	CourierID string
	Period    string // DAILY | WEEKLY | MONTHLY
}

// RevenueFilter holds query parameters for the revenue report.
type RevenueFilter struct {
	Period      string // DAILY | WEEKLY | MONTHLY
	ServiceType string // optional, e.g. REG, EXP
}

// HubPerformanceFilter holds query parameters for the hub performance report.
type HubPerformanceFilter struct {
	HubID  string
	Period string // DAILY | WEEKLY | MONTHLY
}
