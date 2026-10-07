//go:build integration

package migrations

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestCatalogMigration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := postgres.Run(ctx,
		"postgres:18.3-bookworm@sha256:80630f83606d8db77d30b3851b16a9f78be2d0d4dda6f7b82a1fdca5ebe3acba",
		postgres.WithDatabase("migration_test"), postgres.WithUsername("tester"), postgres.WithPassword("test_password"),
		postgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, db)
	if err != nil {
		t.Fatal(err)
	}
	databaseURL, err := db.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	m, err := New(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	if err := m.Up(); !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("second up: %v", err)
	}
	version, dirty, err := m.Version()
	if err != nil || version != 1 || dirty {
		t.Fatalf("version=%d dirty=%v error=%v", version, dirty, err)
	}
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	var category, product int64
	if err := conn.QueryRow(ctx, "INSERT INTO categories(name) VALUES ('category') RETURNING id").Scan(&category); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(ctx, "INSERT INTO products(name,price,category_id) VALUES ('product',0,$1) RETURNING id", category).Scan(&product); err != nil {
		t.Fatal(err)
	}
	if category <= 0 || product <= 0 {
		t.Fatal("nonpositive generated identity")
	}
	for _, query := range []string{
		"INSERT INTO categories(name) VALUES ('category')",
		"INSERT INTO products(name,price,category_id) SELECT name,price,category_id FROM products",
	} {
		if _, err := conn.Exec(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := conn.Exec(ctx, "INSERT INTO stocks(product_id,quantity) VALUES ($1,0)", product); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, sql, code string
		args            []any
	}{
		{"empty category", "INSERT INTO categories(name) VALUES ('')", "23514", nil},
		{"null category", "INSERT INTO categories(name) VALUES (NULL)", "23502", nil},
		{"long category", "INSERT INTO categories(name) VALUES ($1)", "22001", []any{strings.Repeat("あ", 101)}},
		{"negative identity", "INSERT INTO categories(id,name) OVERRIDING SYSTEM VALUE VALUES (-1,'invalid')", "23514", nil},
		{"empty product", "INSERT INTO products(name,price,category_id) VALUES ('',0,$1)", "23514", []any{category}},
		{"long product", "INSERT INTO products(name,price,category_id) VALUES ($1,0,$2)", "22001", []any{strings.Repeat("あ", 201), category}},
		{"negative price", "INSERT INTO products(name,price,category_id) VALUES ('bad',-1,$1)", "23514", []any{category}},
		{"null price", "INSERT INTO products(name,price,category_id) VALUES ('bad',NULL,$1)", "23502", []any{category}},
		{"missing category", "INSERT INTO products(name,price,category_id) VALUES ('bad',0,9223372036854775807)", "23503", nil},
		{"missing product", "INSERT INTO stocks(product_id,quantity) VALUES (9223372036854775807,0)", "23503", nil},
		{"negative stock", "UPDATE stocks SET quantity=-1 WHERE product_id=$1", "23514", []any{product}},
		{"null stock", "UPDATE stocks SET quantity=NULL WHERE product_id=$1", "23502", []any{product}},
		{"duplicate stock", "INSERT INTO stocks(product_id,quantity) VALUES ($1,0)", "23505", []any{product}},
		{"referenced product", "DELETE FROM products WHERE id=$1", "23503", []any{product}},
		{"referenced category", "DELETE FROM categories WHERE id=$1", "23503", []any{category}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := conn.Exec(ctx, tc.sql, tc.args...)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != tc.code {
				t.Fatalf("expected SQLSTATE %s, got %v", tc.code, err)
			}
		})
	}
	var indexes int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM pg_indexes WHERE schemaname='public' AND tablename IN ('categories','products','stocks')").Scan(&indexes); err != nil {
		t.Fatal(err)
	}
	if indexes != 3 {
		t.Fatalf("expected primary-key indexes only, got %d", indexes)
	}
	if err := m.Steps(-1); err != nil {
		t.Fatal(err)
	}
	var tables int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('categories','products','stocks')").Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Fatalf("down left %d tables", tables)
	}
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM categories").Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Fatal("data remained after down/up")
	}
}
