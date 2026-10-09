package health

import (
	"context"
	"encoding/json"
	"github.com/chrptos/j/internal/problem"
	"net/http"
	"time"
)

type Pinger interface {
	Ping(context.Context) error
}

func Handler(db Pinger, timeout time.Duration) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) {
		respond(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			problem.Write(w, r, 503, "DATABASE_UNAVAILABLE", "データベースに接続できません", nil)
			return
		}
		respond(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return mux
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
