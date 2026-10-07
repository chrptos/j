package main

import (
	"strings"
	"testing"
)

func TestInvalidDatabaseConfiguration(t *testing.T) {
	for _, tc := range []struct{ name, value, message string }{
		{"missing", "", "DATABASE_URL is required"},
		{"invalid", "postgres://user:secret@host:invalid/db", "invalid DATABASE_URL"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", tc.value)
			err := run()
			if err == nil || err.Error() != tc.message {
				t.Fatalf("error: %v", err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("credentials exposed")
			}
		})
	}
}
