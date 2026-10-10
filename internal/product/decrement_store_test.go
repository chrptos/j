//go:build integration

package product

import (
	"context"
	"errors"
	"github.com/chrptos/j/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDecrementPostgres(t *testing.T) {
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

	pool, err := pgxpool.New(ctx, uri)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, sql := range []string{
		"INSERT INTO categories(name) VALUES ('Books')",
		"INSERT INTO products(name,price,category_id) VALUES ('A',100,1),('B',100,1)",
		"INSERT INTO stocks(product_id,quantity) VALUES (1,50)",
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	store := Store{Pool: pool, AcquireTimeout: 5 * time.Second, QueryTimeout: 5 * time.Second}
	for _, tc := range []struct {
		id   int64
		code string
	}{{999, "PRODUCT_NOT_FOUND"}, {2, "INTERNAL_ERROR"}} {
		_, err := store.Decrement(ctx, tc.id, 1)
		var e *StockError
		if !errors.As(err, &e) || e.Code != tc.code {
			t.Fatalf("error %v", err)
		}
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 80; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := store.Decrement(ctx, 1, 1)
			if err == nil {
				successes.Add(1)
				if result.RemainingStock < 0 {
					t.Error("negative stock")
				}
				return
			}
			var e *StockError
			if !errors.As(err, &e) || e.Code != "INSUFFICIENT_STOCK" {
				t.Errorf("error %v", err)
			}
		}()
	}
	wg.Wait()
	var remaining int32
	if err := pool.QueryRow(ctx, "SELECT quantity FROM stocks WHERE product_id=1").Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if successes.Load() != 50 || remaining != 0 {
		t.Fatalf("successes %d stock %d", successes.Load(), remaining)
	}
	if _, err := pool.Exec(ctx, "UPDATE stocks SET quantity=10 WHERE product_id=1"); err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, "UPDATE stocks SET quantity=quantity WHERE product_id=1"); err != nil {
		t.Fatal(err)
	}
	limited := store
	limited.QueryTimeout = 30 * time.Millisecond
	_, err = limited.Decrement(ctx, 1, 1)
	var dbErr *DBError
	if !errors.As(err, &dbErr) || dbErr.Code != "DATABASE_TIMEOUT" {
		t.Fatalf("error %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT quantity FROM stocks WHERE product_id=1").Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 10 {
		t.Fatalf("timed out decrement changed stock: %d", remaining)
	}
	_, err = store.Decrement(ctx, 1, 11)
	var stockErr *StockError
	if !errors.As(err, &stockErr) || stockErr.Code != "INSUFFICIENT_STOCK" {
		t.Fatalf("error %v", err)
	}
	result, err := store.Decrement(ctx, 1, 10)
	if err != nil || result.RemainingStock != 0 {
		t.Fatalf("result %+v error %v", result, err)
	}
	if _, err := pool.Exec(ctx, "UPDATE stocks SET quantity=2147483647 WHERE product_id=1"); err != nil {
		t.Fatal(err)
	}
	result, err = store.Decrement(ctx, 1, 2147483647)
	if err != nil || result.RemainingStock != 0 {
		t.Fatalf("maximum quantity: %+v %v", result, err)
	}

}
