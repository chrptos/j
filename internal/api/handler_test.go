package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chrptos/j/internal/problem"
)

func TestRouter(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	})
	handler := Router(ok, ok)
	for _, tc := range []struct {
		method, path string
		status       int
		code         string
	}{
		{"GET", "/products", 200, ""}, {"HEAD", "/products", 200, ""},
		{"POST", "/products", 405, "METHOD_NOT_ALLOWED"},
		{"GET", "/missing", 404, "ENDPOINT_NOT_FOUND"},
		{"POST", "/health/live", 405, "METHOD_NOT_ALLOWED"},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, httptest.NewRequest(tc.method, tc.path, nil))
			if res.Code != tc.status {
				t.Fatalf("status: %d", res.Code)
			}
			if tc.method == "HEAD" && res.Body.Len() != 0 {
				t.Fatal("HEAD has a body")
			}
			if tc.status == 405 && res.Header().Get("Allow") != "GET, HEAD" {
				t.Fatal("missing Allow")
			}
			if tc.code != "" {
				var body problem.Response
				if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Code != tc.code || body.Status != tc.status {
					t.Fatalf("body: %+v", body)
				}
			}
		})
	}
}
