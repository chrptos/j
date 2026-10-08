package problem

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type requestInfo struct {
	id    string
	start time.Time
}
type requestKey struct{}

func WithRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := requestInfo{uuid.NewString(), time.Now()}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestKey{}, info)))
	})
}
