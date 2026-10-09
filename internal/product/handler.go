package product

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/chrptos/j/internal/problem"
)

type Searcher interface {
	Search(context.Context, Query) (Page, error)
}

func Handler(store Searcher) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q, fields := ParseQuery(r.URL.RawQuery)
		if len(fields) > 0 {
			problem.Write(w, r, 400, "INVALID_ARGUMENT", "検索条件が不正です", fields)
			return
		}
		page, err := store.Search(r.Context(), q)
		if err != nil {
			code, status := "INTERNAL_ERROR", 500
			var db *DBError
			if errors.As(err, &db) {
				switch db.Code {
				case "DATABASE_UNAVAILABLE", "DATABASE_TIMEOUT", "TRANSACTION_ABORTED":
					code, status = db.Code, 503
				}
			}
			problem.Write(w, r, status, code, "検索を完了できませんでした", nil)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(page)
	})
}
