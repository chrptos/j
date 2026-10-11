package telemetry

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/felixge/httpsnoop"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type Metrics struct {
	Registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func New(pool *pgxpool.Pool) *Metrics {
	m := &Metrics{
		Registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "j_http_requests_total", Help: "Completed HTTP requests."}, []string{"route", "method", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "j_http_request_duration_seconds", Help: "HTTP request duration in seconds.", Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2, 3, 5, 10}}, []string{"route", "method", "status"}),
	}
	m.Registry.MustRegister(m.requests, m.duration, collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	if pool != nil {
		registerPool(m.Registry, pool)
	}
	return m
}

func (m *Metrics) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed := httpsnoop.CaptureMetrics(next, w, r)
		route := r.Pattern
		if _, pattern, ok := strings.Cut(route, " "); ok {
			route = pattern
		}
		switch route {
		case "/products", "/products/{productId}/stock/decrements", "/health/live", "/health/ready":
		default:
			route = "unmatched"
		}
		method := r.Method
		switch method {
		case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
		default:
			method = "OTHER"
		}
		labels := []string{route, method, strconv.Itoa(observed.Code)}
		m.requests.WithLabelValues(labels...).Inc()
		m.duration.WithLabelValues(labels...).Observe(observed.Duration.Seconds())
	})
}

func registerPool(reg *prometheus.Registry, pool *pgxpool.Pool) {
	gauge := func(name, help string, value func(*pgxpool.Stat) float64) {
		reg.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: name, Help: help}, func() float64 { return value(pool.Stat()) }))
	}
	counter := func(name, help string, value func(*pgxpool.Stat) float64) {
		reg.MustRegister(prometheus.NewCounterFunc(prometheus.CounterOpts{Name: name, Help: help}, func() float64 { return value(pool.Stat()) }))
	}
	gauge("j_db_pool_acquired_connections", "Connections currently acquired.", func(s *pgxpool.Stat) float64 { return float64(s.AcquiredConns()) })
	gauge("j_db_pool_idle_connections", "Idle connections.", func(s *pgxpool.Stat) float64 { return float64(s.IdleConns()) })
	gauge("j_db_pool_max_connections", "Configured maximum connections.", func(s *pgxpool.Stat) float64 { return float64(s.MaxConns()) })
	gauge("j_db_pool_total_connections", "Total connections.", func(s *pgxpool.Stat) float64 { return float64(s.TotalConns()) })
	counter("j_db_pool_acquires_total", "Successful connection acquisitions.", func(s *pgxpool.Stat) float64 { return float64(s.AcquireCount()) })
	counter("j_db_pool_empty_acquires_total", "Successful acquisitions that waited for a connection.", func(s *pgxpool.Stat) float64 { return float64(s.EmptyAcquireCount()) })
	counter("j_db_pool_canceled_acquires_total", "Canceled connection acquisitions.", func(s *pgxpool.Stat) float64 { return float64(s.CanceledAcquireCount()) })
	counter("j_db_pool_acquire_duration_seconds_total", "Total time for successful connection acquisitions.", func(s *pgxpool.Stat) float64 { return s.AcquireDuration().Seconds() })
	counter("j_db_pool_empty_acquire_wait_seconds_total", "Total time waiting for successful acquisitions from an empty pool.", func(s *pgxpool.Stat) float64 { return s.EmptyAcquireWaitTime().Seconds() })
}
