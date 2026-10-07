package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/chrptos/j/internal/database"
	"github.com/chrptos/j/migrations"
	"github.com/golang-migrate/migrate/v4"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 || (args[0] != "up" && args[0] != "down") {
		return errors.New("usage: migrate up|down (down reverts one migration)")
	}
	var cfg database.Connection
	if err := env.Parse(&cfg); err != nil {
		return errors.New("invalid database environment configuration")
	}
	databaseURL, err := cfg.URL()
	if err != nil {
		return err
	}
	m, err := migrations.New(databaseURL)
	if err != nil {
		return errors.New("cannot initialize migration connection")
	}
	defer m.Close()
	if args[0] == "up" {
		err = m.Up()
	} else {
		err = m.Steps(-1)
	}
	if errors.Is(err, migrate.ErrNoChange) {
		slog.Info("no migration changes")
		return nil
	}
	if err != nil {
		// DBエラーに接続情報やデータが含まれる場合があるため、値をログへ出さない。
		var dirty migrate.ErrDirty
		if errors.As(err, &dirty) {
			return errors.New("migration is dirty; inspect database state before repair")
		}
		return errors.New("migration execution failed; inspect database state and migration SQL")
	}
	slog.Info("migration completed", "direction", args[0])
	return nil
}
