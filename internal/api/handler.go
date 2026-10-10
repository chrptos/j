package api

import (
	"net/http"
	"time"

	"github.com/chrptos/j/internal/health"
	"github.com/chrptos/j/internal/problem"
	"github.com/chrptos/j/internal/product"
	"github.com/chrptos/j/internal/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Handler(pool *pgxpool.Pool, healthTimeout, acquireTimeout, queryTimeout time.Duration) http.Handler {
	store := product.Store{Pool: pool, AcquireTimeout: acquireTimeout, QueryTimeout: queryTimeout}
	metrics := telemetry.New(pool)
	mux := http.NewServeMux()
	mux.Handle("/metrics", getOnly(promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{})))
	mux.Handle("/", problem.WithRequest(metrics.Wrap(routes(health.Handler(pool, healthTimeout), product.Handler(store), product.DecrementHandler(store)))))
	return mux
}

func Router(health, products, decrements http.Handler) http.Handler {
	return problem.WithRequest(routes(health, products, decrements))
}

func routes(health, products, decrements http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/health/live", getOnly(health))
	mux.Handle("/health/ready", getOnly(health))
	mux.Handle("/products", getOnly(products))
	mux.Handle("/products/{productId}/stock/decrements", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			problem.Write(w, r, 405, "METHOD_NOT_ALLOWED", "許可されないHTTPメソッドです", nil)
			return
		}
		decrements.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		problem.Write(w, r, 404, "ENDPOINT_NOT_FOUND", "指定されたURLはありません", nil)
	})
	return mux
}

func getOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			problem.Write(w, r, 405, "METHOD_NOT_ALLOWED", "許可されないHTTPメソッドです", nil)
			return
		}
		if r.Method == http.MethodHead {
			w = headWriter{w}
		}
		next.ServeHTTP(w, r)
	})
}

type headWriter struct{ http.ResponseWriter }

func (w headWriter) Write(b []byte) (int, error) { return len(b), nil }
