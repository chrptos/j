package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func exampleEnv(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile("../../.env.example")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if key, value, ok := strings.Cut(line, "="); ok {
			t.Setenv(key, value)
		}
	}
}

func TestConfigurationOverrides(t *testing.T) {
	exampleEnv(t)
	t.Setenv("POSTGRES_PASSWORD", "a@b:c/d?e#f")
	t.Setenv("HTTP_PORT", "18080")
	t.Setenv("DB_MAX_CONNS", "3")
	t.Setenv("HEALTH_DB_TIMEOUT", "250ms")
	t.Setenv("DB_STATEMENT_TIMEOUT", "1500ms")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DB.ConnConfig.Password != "a@b:c/d?e#f" || cfg.DB.MaxConns != 3 || cfg.Addr != ":18080" || cfg.Health != 250*time.Millisecond || cfg.DB.ConnConfig.RuntimeParams["statement_timeout"] != "1500" {
		t.Fatal("configuration override did not apply")
	}
}

func TestInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"POSTGRES_PASSWORD", ""}, {"HTTP_PORT", "65536"}, {"DB_PORT", "secret"},
		{"DB_MAX_CONNS", "0"}, {"DB_MAX_CONNS", "2147483648"}, {"DB_MIN_CONNS", "11"},
		{"HTTP_READ_TIMEOUT", "0s"}, {"DB_CONNECT_TIMEOUT", "secret"}, {"DB_LOCK_TIMEOUT", "500us"}, {"DB_SSLMODE", "secret"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			exampleEnv(t)
			t.Setenv(tc.key, tc.value)
			err := run()
			if err == nil {
				t.Fatal("invalid configuration accepted")
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("input value exposed")
			}
		})
	}
}

func TestMissingConfiguration(t *testing.T) {
	exampleEnv(t)
	original := os.Getenv("DB_MAX_CONNS")
	if err := os.Unsetenv("DB_MAX_CONNS"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Setenv("DB_MAX_CONNS", original) })
	if _, err := loadConfig(); err == nil {
		t.Fatal("missing required variable accepted")
	}
}
