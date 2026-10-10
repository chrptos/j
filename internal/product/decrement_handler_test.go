package product

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chrptos/j/internal/problem"
)

type decrementStub struct {
	calls int
	err   error
}

func (s *decrementStub) Decrement(_ context.Context, id int64, q int32) (StockResult, error) {
	s.calls++
	return StockResult{id, 10 - q}, s.err
}
func TestDecrementValidation(t *testing.T) {
	for _, tc := range []struct {
		id, body, media string
		status          int
	}{
		{"1", `{"quantity":1}`, "application/json", 200},
		{"0", `{"quantity":1}`, "application/json", 400},
		{"abc", `{"quantity":1}`, "application/json", 400},
		{"9223372036854775808", `{"quantity":1}`, "application/json", 400},
		{"1", `{}`, "application/json", 400},
		{"1", `null`, "application/json", 400},
		{"1", `{"quantity":null}`, "application/json", 400},
		{"1", `{"quantity":0}`, "application/json", 400},
		{"1", `{"quantity":-1}`, "application/json", 400},
		{"1", `{"quantity":2147483648}`, "application/json", 400},
		{"1", `{"quantity":1.5}`, "application/json", 400},
		{"1", `{"quantity":"1"}`, "application/json", 400},
		{"1", `{"quantity":1} {}`, "application/json", 400},
		{"1", `{`, "application/json", 400},
		{"1", `{"quantity":1}`, "text/plain", 415},
	} {
		t.Run(tc.id+tc.body+tc.media, func(t *testing.T) {
			stub := &decrementStub{}
			req := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			req.SetPathValue("productId", tc.id)
			req.Header.Set("Content-Type", tc.media)
			res := httptest.NewRecorder()
			DecrementHandler(stub).ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("status %d: %s", res.Code, res.Body.String())
			}
			if tc.status != 200 && stub.calls != 0 {
				t.Fatal("invalid input reached DB")
			}
		})
	}
}
func TestDecrementErrors(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{&StockError{"PRODUCT_NOT_FOUND"}, 404, "PRODUCT_NOT_FOUND"},
		{&StockError{"INSUFFICIENT_STOCK"}, 409, "INSUFFICIENT_STOCK"},
		{&StockError{"INTERNAL_ERROR"}, 500, "INTERNAL_ERROR"},
		{&DBError{"DATABASE_TIMEOUT"}, 503, "DATABASE_TIMEOUT"},
		{&DBError{"DATABASE_UNAVAILABLE"}, 503, "DATABASE_UNAVAILABLE"},
		{&DBError{"TRANSACTION_ABORTED"}, 503, "TRANSACTION_ABORTED"},
	} {
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"quantity":1}`))
		req.SetPathValue("productId", "1")
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		DecrementHandler(&decrementStub{err: tc.err}).ServeHTTP(res, req)
		var body problem.Response
		_ = json.Unmarshal(res.Body.Bytes(), &body)
		if res.Code != tc.status || body.Code != tc.code {
			t.Fatalf("%d %+v", res.Code, body)
		}
	}
}
