package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics 应用指标
type Metrics struct {
	// HTTP请求指标
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// 设备指标
	DevicesTotal       prometheus.Gauge
	DevicesOnline      prometheus.Gauge
	DeviceRegistrations *prometheus.CounterVec

	// 隧道指标
	TunnelsTotal   *prometheus.GaugeVec
	TunnelFailures *prometheus.CounterVec

	// 数据库指标
	DBQueriesTotal    *prometheus.CounterVec
	DBQueryDuration   *prometheus.HistogramVec
	DBConnectionsPool *prometheus.GaugeVec
}

// New 创建指标收集器
func New() *Metrics {
	return &Metrics{
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "edgelink_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "edgelink_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),

		DevicesTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "edgelink_devices_total",
				Help: "Total number of registered devices",
			},
		),

		DevicesOnline: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "edgelink_devices_online",
				Help: "Number of currently online devices",
			},
		),

		DeviceRegistrations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "edgelink_device_registrations_total",
				Help: "Total number of device registrations",
			},
			[]string{"platform", "status"},
		),

		TunnelsTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "edgelink_tunnels_total",
				Help: "Total number of active tunnels",
			},
			[]string{"connection_type"},
		),

		TunnelFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "edgelink_tunnel_failures_total",
				Help: "Total number of tunnel establishment failures",
			},
			[]string{"reason"},
		),

		DBQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "edgelink_db_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation", "table"},
		),

		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "edgelink_db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "table"},
		),

		DBConnectionsPool: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "edgelink_db_connections_pool",
				Help: "Database connection pool statistics",
			},
			[]string{"state"}, // "idle", "in_use", "total"
		),
	}
}

// RecordHTTPRequest 记录HTTP请求
func (m *Metrics) RecordHTTPRequest(method, path, status string, duration float64) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
}

// RecordDeviceRegistration 记录设备注册
func (m *Metrics) RecordDeviceRegistration(platform, status string) {
	m.DeviceRegistrations.WithLabelValues(platform, status).Inc()
}

// UpdateDeviceCount 更新设备数量
func (m *Metrics) UpdateDeviceCount(total, online int) {
	m.DevicesTotal.Set(float64(total))
	m.DevicesOnline.Set(float64(online))
}

// UpdateTunnelCount 更新隧道数量
func (m *Metrics) UpdateTunnelCount(connType string, count int) {
	m.TunnelsTotal.WithLabelValues(connType).Set(float64(count))
}

// RecordTunnelFailure 记录隧道失败
func (m *Metrics) RecordTunnelFailure(reason string) {
	m.TunnelFailures.WithLabelValues(reason).Inc()
}
