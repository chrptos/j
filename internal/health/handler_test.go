package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type pingFunc func(context.Context) error

func (f pingFunc) Ping(ctx context.Context) error { return f(ctx) }

func TestLiveDoesNotRequireDatabase(t *testing.T) {
	handler := Handler(pingFunc(func(context.Context) error {
		t.Fatal("liveness must not access database")
		return nil
	}))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status: %d", res.Code)
	}
}

func TestReady(t *testing.T) {
	for _, tc := range []struct {
		name        string
		err         error
		status      int
		contentType string
	}{
		{"connected", nil, 200, "application/json"},
		{"disconnected", errors.New("secret connection information"), 503, "application/problem+json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := Handler(pingFunc(func(ctx context.Context) error {
				called = true
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("missing deadline")
				}
				return tc.err
			}))
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
			if !called || res.Code != tc.status {
				t.Fatalf("called=%v status=%d", called, res.Code)
			}
			if got := res.Header().Get("Content-Type"); got != tc.contentType {
				t.Fatalf("Content-Type: %s", got)
			}
			if strings.Contains(res.Body.String(), "secret") {
				t.Fatal("database error exposed")
			}
		})
	}
}

func TestReadyPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	handler := Handler(pingFunc(func(ctx context.Context) error { return ctx.Err() }))
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil).WithContext(ctx)
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: %d", res.Code)
	}
}
