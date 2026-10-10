package product

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type StockResult struct {
	ProductID      int64 `json:"productId"`
	RemainingStock int32 `json:"remainingStock"`
}

type StockError struct{ Code string }

func (e *StockError) Error() string { return e.Code }

func (s Store) Decrement(ctx context.Context, id int64, quantity int32) (StockResult, error) {
	result := StockResult{ProductID: id}
	acquire, cancel := context.WithTimeout(ctx, s.AcquireTimeout)
	conn, err := s.Pool.Acquire(acquire)
	cancel()
	if err != nil {
		return result, &DBError{Code: "DATABASE_UNAVAILABLE"}
	}
	defer conn.Release()
	query, cancel := context.WithTimeout(ctx, s.QueryTimeout)
	defer cancel()
	tx, err := conn.BeginTx(query, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return result, classifyDB(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	err = tx.QueryRow(query, `UPDATE stocks SET quantity = quantity - $2 WHERE product_id = $1 AND quantity >= $2 RETURNING quantity`, id, quantity).Scan(&result.RemainingStock)
	if errors.Is(err, pgx.ErrNoRows) {
		var productExists, stockExists bool
		err = tx.QueryRow(query, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1), EXISTS(SELECT 1 FROM stocks WHERE product_id=$1)`, id).Scan(&productExists, &stockExists)
		if err != nil {
			return result, classifyDB(err)
		}
		code := "INSUFFICIENT_STOCK"
		if !productExists {
			code = "PRODUCT_NOT_FOUND"
		} else if !stockExists {
			code = "INTERNAL_ERROR"
		}
		return result, &StockError{Code: code}
	}
	if err != nil {
		return result, classifyDB(err)
	}
	if err = tx.Commit(query); err != nil {
		return result, classifyDB(err)
	}
	return result, nil
}
