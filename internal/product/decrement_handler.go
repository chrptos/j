package product

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime"
	"net/http"
	"strconv"

	"github.com/chrptos/j/internal/problem"
)

type Decrementer interface {
	Decrement(context.Context, int64, int32) (StockResult, error)
}

func DecrementHandler(store Decrementer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || media != "application/json" {
			problem.Write(w, r, 415, "UNSUPPORTED_MEDIA_TYPE", "application/jsonを指定してください", nil)
			return
		}
		id, err := strconv.ParseInt(r.PathValue("productId"), 10, 64)
		fields := []problem.FieldError{}
		if err != nil || id <= 0 {
			fields = append(fields, problem.FieldError{Field: "productId", Code: "INVALID_ARGUMENT", Message: "正の整数を指定してください"})
		}
		var input struct {
			Quantity *int64 `json:"quantity"`
		}
		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&input)
		var extra any
		if err != nil || decoder.Decode(&extra) != io.EOF {
			fields = append(fields, problem.FieldError{Field: "body", Code: "INVALID_FORMAT", Message: "JSONが不正です"})
		} else if input.Quantity == nil || *input.Quantity < 1 || *input.Quantity > math.MaxInt32 {
			fields = append(fields, problem.FieldError{Field: "quantity", Code: "INVALID_ARGUMENT", Message: "1〜2147483647の整数を指定してください"})
		}
		if len(fields) > 0 {
			problem.Write(w, r, 400, "INVALID_ARGUMENT", "減算条件が不正です", fields)
			return
		}
		result, err := store.Decrement(r.Context(), id, int32(*input.Quantity))
		if err != nil {
			code, status, detail := "INTERNAL_ERROR", 500, "在庫減算を完了できませんでした"
			var stock *StockError
			var db *DBError
			if errors.As(err, &stock) {
				switch stock.Code {
				case "PRODUCT_NOT_FOUND":
					code, status, detail = stock.Code, 404, "指定された商品はありません"
				case "INSUFFICIENT_STOCK":
					code, status, detail = stock.Code, 409, "指定された数量の在庫がありません"
				}
			} else if errors.As(err, &db) {
				switch db.Code {
				case "DATABASE_UNAVAILABLE", "DATABASE_TIMEOUT", "TRANSACTION_ABORTED":
					code, status = db.Code, 503
				}
			}
			problem.Write(w, r, status, code, detail, nil)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})
}
