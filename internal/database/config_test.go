package database

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConnectionURL(t *testing.T) {
	cfg := Connection{Host: "::1", Port: 5432, User: "user@name", Password: "p@ss:/?#", Name: "catalog", SSLMode: "disable"}
	uri, err := cfg.URL()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := pgxpool.ParseConfig(uri)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.ConnConfig.Host != cfg.Host || parsed.ConnConfig.User != cfg.User || parsed.ConnConfig.Password != cfg.Password {
		t.Fatal("connection fields changed during URL encoding")
	}
}
