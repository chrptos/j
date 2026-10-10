package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chrptos/j/internal/problem"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHTTPMetrics(t *testing.T) {
	m := New(nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/products/{productId}/stock/decrements", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(409) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) })
	handler := problem.WithRequest(m.Wrap(mux))
	for _, path := range []string{"/products/1/stock/decrements", "/products/999/stock/decrements", "/unknown-a", "/unknown-b"} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest("POST", path, nil))
	}
	metrics, err := m.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	foundCount, foundHistogram := false, false
	for _, family := range metrics {
		if family.GetName() != "j_http_requests_total" && family.GetName() != "j_http_request_duration_seconds" {
			continue
		}
		if len(family.Metric) != 2 {
			t.Fatalf("unbounded labels: %+v", family)
		}
		for _, sample := range family.Metric {
			var route string
			for _, label := range sample.Label {
				if label.GetName() == "route" {
					route = label.GetValue()
				}
			}
			if route != "/products/{productId}/stock/decrements" && route != "unmatched" {
				t.Fatalf("route %q", route)
			}
			if family.GetName() == "j_http_requests_total" {
				foundCount = true
				if sample.Counter.GetValue() != 2 {
					t.Fatal("wrong request count")
				}
			}
			if family.GetName() == "j_http_request_duration_seconds" {
				foundHistogram = true
				if sample.Histogram.GetSampleCount() != 2 {
					t.Fatal("wrong duration count")
				}
			}
		}
	}
	if !foundCount || !foundHistogram {
		t.Fatal("missing HTTP metrics")
	}
}

func TestPoolMetrics(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://tester@localhost/test?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	cfg.MinConns = 0
	cfg.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	families, err := New(pool).Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() == "j_db_pool_max_connections" {
			if family.Metric[0].Gauge.GetValue() != 4 {
				t.Fatal("wrong pool limit")
			}
			return
		}
	}
	t.Fatal("pool metric missing")
}
