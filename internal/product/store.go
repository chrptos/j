package product

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Item struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Price        int32  `json:"price"`
	CategoryName string `json:"categoryName"`
}

type Page struct {
	Items   []Item `json:"items"`
	Limit   int    `json:"limit"`
	Offset  int64  `json:"offset"`
	HasNext bool   `json:"hasNext"`
}

type DBError struct{ Code string }

func (e *DBError) Error() string { return e.Code }

type Store struct {
	Pool                         *pgxpool.Pool
	AcquireTimeout, QueryTimeout time.Duration
}

func (s Store) Search(ctx context.Context, q Query) (Page, error) {
	page := Page{Items: []Item{}, Limit: q.Limit, Offset: q.Offset}
	acquire, cancel := context.WithTimeout(ctx, s.AcquireTimeout)
	conn, err := s.Pool.Acquire(acquire)
	cancel()
	if err != nil {
		return page, &DBError{Code: "DATABASE_UNAVAILABLE"}
	}
	defer conn.Release()
	query, cancel := context.WithTimeout(ctx, s.QueryTimeout)
	defer cancel()
	sql, args := searchSQL(q)
	rows, err := conn.Query(query, sql, args...)
	if err != nil {
		return page, classifyDB(err)
	}
	defer rows.Close()
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.CategoryName); err != nil {
			return page, classifyDB(err)
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return page, classifyDB(err)
	}
	if len(page.Items) > q.Limit {
		page.HasNext = true
		page.Items = page.Items[:q.Limit]
	}
	return page, nil
}

func searchSQL(q Query) (string, []any) {
	clauses := []string{}
	args := []any{}
	bind := func(value any) string { args = append(args, value); return fmt.Sprintf("$%d", len(args)) }
	if q.Keyword != "" {
		literal := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q.Keyword)
		clauses = append(clauses, `p.name LIKE `+bind("%"+literal+"%")+` ESCAPE E'\\'`)
	}
	if q.CategoryID != nil {
		clauses = append(clauses, "p.category_id = "+bind(*q.CategoryID))
	}
	if q.MinPrice != nil {
		clauses = append(clauses, "p.price >= "+bind(*q.MinPrice))
	}
	if q.MaxPrice != nil {
		clauses = append(clauses, "p.price <= "+bind(*q.MaxPrice))
	}
	sql := "SELECT p.id, p.name, p.price, c.name FROM products p JOIN categories c ON c.id = p.category_id"
	if len(clauses) > 0 {
		sql += " WHERE " + strings.Join(clauses, " AND ")
	}
	order := "p.id ASC"
	switch q.Sort {
	case "price_asc":
		order = "p.price ASC, p.id ASC"
	case "price_desc":
		order = "p.price DESC, p.id ASC"
	}
	sql += " ORDER BY " + order + " LIMIT " + bind(q.Limit+1) + " OFFSET " + bind(q.Offset)
	return sql, args
}

func classifyDB(err error) error {
	var pgErr *pgconn.PgError
	var netErr net.Error
	var connect *pgconn.ConnectError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "57014", "55P03":
			return &DBError{Code: "DATABASE_TIMEOUT"}
		case "40P01", "40001":
			return &DBError{Code: "TRANSACTION_ABORTED"}
		}
		if strings.HasPrefix(pgErr.Code, "08") || pgErr.Code == "57P01" || pgErr.Code == "57P02" || pgErr.Code == "57P03" {
			return &DBError{Code: "DATABASE_UNAVAILABLE"}
		}
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || pgconn.Timeout(err) {
		return &DBError{Code: "DATABASE_TIMEOUT"}
	}
	if errors.As(err, &netErr) || errors.As(err, &connect) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return &DBError{Code: "DATABASE_UNAVAILABLE"}
	}
	return &DBError{Code: "INTERNAL_ERROR"}
}
