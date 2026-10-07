package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/chrptos/j/internal/health"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		slog.Error("API stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.NewWithConfig(ctx, cfg.DB)
	if err != nil {
		return errors.New("cannot initialize database pool")
	}
	defer pool.Close()
	startup, cancel := context.WithTimeout(ctx, cfg.Startup)
	err = pool.Ping(startup)
	cancel()
	if err != nil {
		return errors.New("database startup check failed")
	}
	server := &http.Server{
		Addr: cfg.Addr, Handler: health.Handler(pool, cfg.Health),
		ReadHeaderTimeout: cfg.ReadHeader, ReadTimeout: cfg.Read,
		WriteTimeout: cfg.Write, IdleTimeout: cfg.Idle,
	}
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	slog.Info("API started", "addr", cfg.Addr)
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server: %w", err)
		}
		return nil
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), cfg.Shutdown)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			return fmt.Errorf("HTTP shutdown: %w", err)
		}
		return nil
	}
}
