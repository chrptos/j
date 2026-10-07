package product

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestDatabaseErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code string
	}{
		{&pgconn.PgError{Code: "57014"}, "DATABASE_TIMEOUT"},
		{&pgconn.PgError{Code: "55P03"}, "DATABASE_TIMEOUT"},
		{&pgconn.PgError{Code: "40P01"}, "TRANSACTION_ABORTED"},
		{&pgconn.PgError{Code: "40001"}, "TRANSACTION_ABORTED"},
		{&pgconn.PgError{Code: "08006"}, "DATABASE_UNAVAILABLE"},
		{&pgconn.PgError{Code: "57P01"}, "DATABASE_UNAVAILABLE"},
		{context.DeadlineExceeded, "DATABASE_TIMEOUT"},
		{&pgconn.PgError{Code: "42P01"}, "INTERNAL_ERROR"},
		{errors.New("unexpected"), "INTERNAL_ERROR"},
	} {
		var db *DBError
		err := classifyDB(fmt.Errorf("wrapped: %w", tc.err))
		if !errors.As(err, &db) || db.Code != tc.code {
			t.Fatalf("expected %s, got %v", tc.code, err)
		}
	}
}
