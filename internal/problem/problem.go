package problem

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Response struct {
	Type     string       `json:"type"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail"`
	Instance string       `json:"instance"`
	Code     string       `json:"code"`
	TraceID  string       `json:"traceId"`
	Errors   []FieldError `json:"errors,omitempty"`
}

func Write(w http.ResponseWriter, r *http.Request, status int, code, detail string, fields []FieldError) {
	info, ok := r.Context().Value(requestKey{}).(requestInfo)
	if !ok {
		info = requestInfo{uuid.NewString(), time.Now()}
	}
	body := Response{
		Type:  "https://github.com/chrptos/j/blob/develop/docs/design/errors.md#" + strings.ReplaceAll(strings.ToLower(code), "_", "-"),
		Title: http.StatusText(status), Status: status, Detail: detail, Code: code,
		Instance: "urn:uuid:" + info.id, TraceID: info.id, Errors: fields,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
	if status >= 500 {
		slog.Error("request failed", "traceId", info.id, "api", r.URL.Path, "status", status, "code", code, "duration", time.Since(info.start))
	}
}
