package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/chrptos/j/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

type settings struct {
	HTTPPort   int `env:"HTTP_PORT,required"`
	Connection database.Connection
	MaxConns   int32         `env:"DB_MAX_CONNS,required"`
	MinConns   int32         `env:"DB_MIN_CONNS,required"`
	ReadHeader time.Duration `env:"HTTP_READ_HEADER_TIMEOUT,required"`
	Read       time.Duration `env:"HTTP_READ_TIMEOUT,required"`
	Write      time.Duration `env:"HTTP_WRITE_TIMEOUT,required"`
	Idle       time.Duration `env:"HTTP_IDLE_TIMEOUT,required"`
	Shutdown   time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT,required"`
	Startup    time.Duration `env:"DB_STARTUP_TIMEOUT,required"`
	Acquire    time.Duration `env:"DB_ACQUIRE_TIMEOUT,required"`
	Query      time.Duration `env:"DB_QUERY_TIMEOUT,required"`
	Health     time.Duration `env:"HEALTH_DB_TIMEOUT,required"`
	Connect    time.Duration `env:"DB_CONNECT_TIMEOUT,required"`
	Lock       time.Duration `env:"DB_LOCK_TIMEOUT,required"`
	Statement  time.Duration `env:"DB_STATEMENT_TIMEOUT,required"`
}

type config struct {
	settings
	DB   *pgxpool.Config
	Addr string
}

func loadConfig() (config, error) {
	var c config
	if err := env.Parse(&c.settings); err != nil {
		// 型変換のエラーには入力値が含まれるため、そのままログへ出さない。
		var parse env.ParseError
		var missing env.VarIsNotSetError
		var empty env.EmptyVarError
		switch {
		case errors.As(err, &parse):
			return c, fmt.Errorf("invalid configuration field: %s", parse.Name)
		case errors.As(err, &missing):
			return c, fmt.Errorf("%s is required", missing.Key)
		case errors.As(err, &empty):
			return c, fmt.Errorf("%s must not be empty", empty.Key)
		default:
			return c, errors.New("invalid environment configuration")
		}
	}
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return c, errors.New("HTTP_PORT must be between 1 and 65535")
	}
	if c.MaxConns < 1 || c.MinConns < 0 || c.MinConns > c.MaxConns {
		return c, errors.New("DB connection counts must satisfy 0 <= DB_MIN_CONNS <= DB_MAX_CONNS and DB_MAX_CONNS >= 1")
	}
	for key, duration := range map[string]time.Duration{
		"HTTP_READ_HEADER_TIMEOUT": c.ReadHeader, "HTTP_READ_TIMEOUT": c.Read,
		"HTTP_WRITE_TIMEOUT": c.Write, "HTTP_IDLE_TIMEOUT": c.Idle,
		"HTTP_SHUTDOWN_TIMEOUT": c.Shutdown, "DB_STARTUP_TIMEOUT": c.Startup,
		"HEALTH_DB_TIMEOUT": c.Health, "DB_ACQUIRE_TIMEOUT": c.Acquire, "DB_QUERY_TIMEOUT": c.Query, "DB_CONNECT_TIMEOUT": c.Connect,
		"DB_LOCK_TIMEOUT": c.Lock, "DB_STATEMENT_TIMEOUT": c.Statement,
	} {
		if duration < time.Millisecond {
			return c, fmt.Errorf("%s must be at least 1ms", key)
		}
	}
	c.Addr = net.JoinHostPort("", strconv.Itoa(c.HTTPPort))
	databaseURL, err := c.Connection.URL()
	if err != nil {
		return c, err
	}
	db, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return c, errors.New("invalid database configuration")
	}
	db.MaxConns, db.MinConns = c.MaxConns, c.MinConns
	db.ConnConfig.ConnectTimeout = c.Connect
	db.ConnConfig.RuntimeParams["lock_timeout"] = strconv.FormatInt(c.Lock.Milliseconds(), 10)
	db.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(c.Statement.Milliseconds(), 10)
	c.DB = db
	return c, nil
}
