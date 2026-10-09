package product

import (
	"math"
	"net/url"
	"strconv"
	"unicode/utf8"

	"github.com/chrptos/j/internal/problem"
)

type Query struct {
	Keyword            string
	CategoryID         *int64
	MinPrice, MaxPrice *int32
	Sort               string
	Limit              int
	Offset             int64
}

func ParseQuery(raw string) (Query, []problem.FieldError) {
	q := Query{Sort: "id_asc", Limit: 20}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return q, []problem.FieldError{{Field: "query", Code: "INVALID_FORMAT", Message: "クエリ文字列が不正です"}}
	}
	fields := []problem.FieldError{}
	invalid := func(field string) {
		fields = append(fields, problem.FieldError{Field: field, Code: "INVALID_ARGUMENT", Message: "型または範囲が不正です"})
	}
	for _, key := range []string{"keyword", "categoryId", "minPrice", "maxPrice", "sort", "limit", "offset"} {
		if len(values[key]) > 1 {
			invalid(key)
		}
	}
	q.Keyword = values.Get("keyword")
	if !utf8.ValidString(q.Keyword) {
		invalid("keyword")
	}
	number := func(key string, min, max int64) *int64 {
		if _, ok := values[key]; !ok {
			return nil
		}
		n, err := strconv.ParseInt(values.Get(key), 10, 64)
		if err != nil || n < min || n > max {
			invalid(key)
			return nil
		}
		return &n
	}
	q.CategoryID = number("categoryId", 1, math.MaxInt64)
	if n := number("minPrice", 0, math.MaxInt32); n != nil {
		v := int32(*n)
		q.MinPrice = &v
	}
	if n := number("maxPrice", 0, math.MaxInt32); n != nil {
		v := int32(*n)
		q.MaxPrice = &v
	}
	if n := number("limit", 1, 100); n != nil {
		q.Limit = int(*n)
	}
	if n := number("offset", 0, math.MaxInt64); n != nil {
		q.Offset = *n
	}
	if value, ok := values["sort"]; ok {
		switch value[0] {
		case "id_asc", "price_asc", "price_desc":
			q.Sort = value[0]
		default:
			invalid("sort")
		}
	}
	if q.MinPrice != nil && q.MaxPrice != nil && *q.MinPrice > *q.MaxPrice {
		invalid("minPrice")
	}
	return q, fields
}
