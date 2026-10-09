package database

import (
	"errors"
	"net"
	"net/url"
	"strconv"
)

// Connection is shared by the API and migration command.
type Connection struct {
	Host     string `env:"DB_HOST,required,notEmpty"`
	Port     int    `env:"DB_PORT,required"`
	User     string `env:"POSTGRES_USER,required,notEmpty"`
	Password string `env:"POSTGRES_PASSWORD,required,notEmpty"`
	Name     string `env:"POSTGRES_DB,required,notEmpty"`
	SSLMode  string `env:"DB_SSLMODE,required,notEmpty"`
}

func (c Connection) URL() (string, error) {
	if c.Port < 1 || c.Port > 65535 {
		return "", errors.New("DB_PORT must be between 1 and 65535")
	}
	u := url.URL{Scheme: "postgres", Host: net.JoinHostPort(c.Host, strconv.Itoa(c.Port)), User: url.UserPassword(c.User, c.Password), Path: "/" + c.Name}
	u.RawQuery = url.Values{"sslmode": {c.SSLMode}}.Encode()
	return u.String(), nil
}
