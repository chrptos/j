package product

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chrptos/j/internal/problem"
)

type searchFunc func(context.Context, Query) (Page, error)

func (f searchFunc) Search(ctx context.Context, q Query) (Page, error) { return f(ctx, q) }

func TestSearchRejectsInvalidQueryBeforeDatabase(t *testing.T) {
	for _, raw := range []string{
		"limit=0", "limit=101", "limit=abc", "offset=-1", "offset=9223372036854775808",
		"categoryId=0", "categoryId=9223372036854775808", "categoryId=",
		"minPrice=-1", "maxPrice=2147483648", "minPrice=20&maxPrice=10",
		"sort=price", "sort=", "limit=1&limit=2", "keyword=%xx", "keyword=%FF",
	} {
		t.Run(raw, func(t *testing.T) {
			handler := Handler(searchFunc(func(context.Context, Query) (Page, error) {
				t.Fatal("invalid input reached database")
				return Page{}, nil
			}))
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/products?"+raw, nil))
			var body problem.Response
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if res.Code != 400 || body.Code != "INVALID_ARGUMENT" || len(body.Errors) == 0 || body.TraceID == "" || body.Instance != "urn:uuid:"+body.TraceID {
				t.Fatalf("response: %s", res.Body.String())
			}
			if res.Header().Get("Content-Type") != "application/problem+json" {
				t.Fatal("wrong content type")
			}
		})
	}
}

func TestQueryDefaultsAndBoundaries(t *testing.T) {
	q, fields := ParseQuery("")
	if len(fields) != 0 || q.Limit != 20 || q.Offset != 0 || q.Sort != "id_asc" {
		t.Fatalf("defaults: %+v, %v", q, fields)
	}
	q, fields = ParseQuery("categoryId=9223372036854775807&minPrice=0&maxPrice=2147483647&offset=9223372036854775807&limit=100")
	if len(fields) != 0 || q.Limit != 100 || q.Offset != 9223372036854775807 || *q.MaxPrice != 2147483647 {
		t.Fatalf("boundaries: %+v, %v", q, fields)
	}
}

func TestSearchFailureResponse(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{&DBError{Code: "DATABASE_UNAVAILABLE"}, 503, "DATABASE_UNAVAILABLE"},
		{&DBError{Code: "DATABASE_TIMEOUT"}, 503, "DATABASE_TIMEOUT"},
		{&DBError{Code: "TRANSACTION_ABORTED"}, 503, "TRANSACTION_ABORTED"},
		{errors.New("secret database internals"), 500, "INTERNAL_ERROR"},
	} {
		handler := Handler(searchFunc(func(context.Context, Query) (Page, error) { return Page{}, tc.err }))
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest("GET", "/products", nil))
		var body problem.Response
		if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if res.Code != tc.status || body.Code != tc.code || strings.Contains(res.Body.String(), "secret") {
			t.Fatalf("response: %s", res.Body.String())
		}
	}
}
