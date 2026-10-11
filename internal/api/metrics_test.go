package api

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetricsEndpoint(t *testing.T) {
	handler := Handler(nil, time.Second, time.Second, time.Second)
	for i := 0; i < 2; i++ {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest("GET", "/metrics", nil))
		if res.Code != 200 || !strings.Contains(res.Body.String(), "go_goroutines") {
			t.Fatalf("metrics %d", res.Code)
		}
		if strings.Contains(res.Body.String(), "j_http_requests_total{") {
			t.Fatal("scrape counted as API traffic")
		}
	}
}
