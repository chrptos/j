//go:build integration

package product

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/chrptos/j/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestSearchPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := postgres.Run(ctx, "postgres:18.3-bookworm@sha256:80630f83606d8db77d30b3851b16a9f78be2d0d4dda6f7b82a1fdca5ebe3acba", postgres.WithDatabase("search_test"), postgres.WithUsername("tester"), postgres.WithPassword("test_password"), postgres.BasicWaitStrategies())
	testcontainers.CleanupContainer(t, db)
	if err != nil {
		t.Fatal(err)
	}
	uri, err := db.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	m, err := migrations.New(uri)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	cfg, err := pgxpool.ParseConfig(uri)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	for _, sql := range []string{
		"INSERT INTO categories(name) VALUES ('Books'),('Food')",
		`INSERT INTO products(name,price,category_id) VALUES ('Go入門',200,1),('go入門',100,1),('Go_100%\guide',200,1),('Rice',0,2),('Go上級',300,1)`,
		"INSERT INTO stocks(product_id,quantity) SELECT id,100 FROM products",
	} {
		if _, err := tx.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	store := Store{Pool: pool, AcquireTimeout: time.Second, QueryTimeout: time.Second}
	for _, tc := range []struct {
		name, raw string
		ids       []int64
		next      bool
	}{
		{"default", "", []int64{1, 2, 3, 4, 5}, false},
		{"ascending ties", "sort=price_asc", []int64{4, 2, 1, 3, 5}, false},
		{"descending ties", "sort=price_desc", []int64{5, 1, 3, 2, 4}, false},
		{"keyword case", "keyword=Go", []int64{1, 3, 5}, false},
		{"lowercase", "keyword=go", []int64{2}, false},
		{"underscore literal", "keyword=_", []int64{3}, false},
		{"percent literal", "keyword=%25", []int64{3}, false},
		{"backslash literal", "keyword=%5C", []int64{3}, false},
		{"category", "categoryId=2", []int64{4}, false},
		{"price boundaries", "minPrice=200&maxPrice=200", []int64{1, 3}, false},
		{"combined", "keyword=Go&categoryId=1&minPrice=200&maxPrice=200", []int64{1, 3}, false},
		{"missing category", "categoryId=999", []int64{}, false},
		{"first page", "limit=2", []int64{1, 2}, true},
		{"middle page", "limit=2&offset=2", []int64{3, 4}, true},
		{"last page", "limit=2&offset=4", []int64{5}, false},
		{"exact page", "limit=5", []int64{1, 2, 3, 4, 5}, false},
		{"past last page", "offset=9223372036854775807", []int64{}, false},
		{"SQL injection literal", "keyword=%27+OR+1%3D1+--", []int64{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, fields := ParseQuery(tc.raw)
			if len(fields) > 0 {
				t.Fatal(fields)
			}
			res := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/products?"+tc.raw, nil).WithContext(ctx)
			Handler(store).ServeHTTP(res, req)
			if res.Code != 200 {
				t.Fatalf("HTTP %d: %s", res.Code, res.Body.String())
			}
			var page Page
			if err := json.Unmarshal(res.Body.Bytes(), &page); err != nil {
				t.Fatal(err)
			}
			ids := []int64{}
			for _, item := range page.Items {
				ids = append(ids, item.ID)
				if item.CategoryName == "" {
					t.Fatal("missing joined category")
				}
			}
			if !reflect.DeepEqual(ids, tc.ids) || page.HasNext != tc.next || page.Items == nil || page.Limit != q.Limit || page.Offset != q.Offset {
				t.Fatalf("page=%+v ids=%v", page, ids)
			}
		})
	}
	t.Run("pool exhausted", func(t *testing.T) {
		held, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer held.Release()
		limited := store
		limited.AcquireTimeout = 20 * time.Millisecond
		_, err = limited.Search(ctx, Query{Limit: 20, Sort: "id_asc"})
		var dbErr *DBError
		if !errors.As(err, &dbErr) || dbErr.Code != "DATABASE_UNAVAILABLE" {
			t.Fatalf("error: %v", err)
		}
	})
	t.Run("query timeout", func(t *testing.T) {
		locker, err := pgxpool.New(ctx, uri)
		if err != nil {
			t.Fatal(err)
		}
		defer locker.Close()
		lock, err := locker.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer lock.Rollback(context.Background())
		if _, err := lock.Exec(ctx, "LOCK products IN ACCESS EXCLUSIVE MODE"); err != nil {
			t.Fatal(err)
		}
		limited := store
		limited.QueryTimeout = 20 * time.Millisecond
		_, err = limited.Search(ctx, Query{Limit: 20, Sort: "id_asc"})
		var dbErr *DBError
		if !errors.As(err, &dbErr) || dbErr.Code != "DATABASE_TIMEOUT" {
			t.Fatalf("error: %v", err)
		}
	})
}
