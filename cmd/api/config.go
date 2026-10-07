package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type config struct {
	DB                                                       *pgxpool.Config
	Addr                                                     string
	ReadHeader, Read, Write, Idle, Shutdown, Startup, Health time.Duration
}

func loadConfig() (config, error) {
	var c config
	values := make(map[string]string)
	for _, key := range []string{"HTTP_PORT", "DB_HOST", "DB_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB", "DB_SSLMODE"} {
		value := os.Getenv(key)
		if value == "" {
			return c, fmt.Errorf("%s is required", key)
		}
		values[key] = value
	}
	for _, key := range []string{"HTTP_PORT", "DB_PORT"} {
		port, err := strconv.Atoi(values[key])
		if err != nil || port < 1 || port > 65535 {
			return c, fmt.Errorf("%s must be a port between 1 and 65535", key)
		}
	}
	c.Addr = net.JoinHostPort("", values["HTTP_PORT"])
	u := url.URL{Scheme: "postgres", Host: net.JoinHostPort(values["DB_HOST"], values["DB_PORT"]), User: url.UserPassword(values["POSTGRES_USER"], values["POSTGRES_PASSWORD"]), Path: "/" + values["POSTGRES_DB"]}
	q := url.Values{"sslmode": {values["DB_SSLMODE"]}}
	u.RawQuery = q.Encode()
	db, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		return c, fmt.Errorf("invalid database configuration")
	}
	c.DB = db
	for _, setting := range []struct {
		key     string
		target  *int32
		minimum int64
	}{
		{"DB_MAX_CONNS", &db.MaxConns, 1}, {"DB_MIN_CONNS", &db.MinConns, 0},
	} {
		n, err := strconv.ParseInt(os.Getenv(setting.key), 10, 32)
		if err != nil || n < setting.minimum {
			return c, fmt.Errorf("%s is invalid", setting.key)
		}
		*setting.target = int32(n)
	}
	if db.MinConns > db.MaxConns {
		return c, fmt.Errorf("DB_MIN_CONNS must not exceed DB_MAX_CONNS")
	}
	var lock, statement time.Duration
	for _, setting := range []struct {
		key    string
		target *time.Duration
	}{
		{"HTTP_READ_HEADER_TIMEOUT", &c.ReadHeader}, {"HTTP_READ_TIMEOUT", &c.Read},
		{"HTTP_WRITE_TIMEOUT", &c.Write}, {"HTTP_IDLE_TIMEOUT", &c.Idle},
		{"HTTP_SHUTDOWN_TIMEOUT", &c.Shutdown}, {"DB_STARTUP_TIMEOUT", &c.Startup},
		{"HEALTH_DB_TIMEOUT", &c.Health}, {"DB_CONNECT_TIMEOUT", &db.ConnConfig.ConnectTimeout},
		{"DB_LOCK_TIMEOUT", &lock}, {"DB_STATEMENT_TIMEOUT", &statement},
	} {
		duration, err := time.ParseDuration(os.Getenv(setting.key))
		if err != nil || duration < time.Millisecond {
			return c, fmt.Errorf("%s must be at least 1ms", setting.key)
		}
		*setting.target = duration
	}
	db.ConnConfig.RuntimeParams["lock_timeout"] = strconv.FormatInt(lock.Milliseconds(), 10)
	db.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(statement.Milliseconds(), 10)
	return c, nil
}
