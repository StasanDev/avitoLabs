package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	LogLevel        string
	ShutdownTimeout time.Duration
	IdempotencyTTL  time.Duration
	Http            HttpConfig
	DB              DBConfig
}

type HttpConfig struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type DBConfig struct {
	Url             string
	MaxConns        int32
	MinConns        int32
	ConnectTimeout  time.Duration
	QueryTimeout    time.Duration
	MaxConnLifetime time.Duration
}

var requiredEnvNames = []string{
	"LOG_LEVEL", "SHUTDOWN_TIMEOUT", "IDEMPOTENCY_TTL", "HTTP_ADDR", "HTTP_READ_TIMEOUT",
	"HTTP_READ_HEADER_TIMEOUT", "HTTP_WRITE_TIMEOUT", "HTTP_IDLE_TIMEOUT",
	"DATABASE_URL", "DATABASE_MIN_CONNS", "DATABASE_MAX_CONNS",
	"DATABASE_CONNECT_TIMEOUT", "DATABASE_QUERY_TIMEOUT", "DATABASE_MAX_CONN_LIFETIME",
}

var durationEnvNames = []string{
	"SHUTDOWN_TIMEOUT", "IDEMPOTENCY_TTL", "HTTP_READ_TIMEOUT",
	"HTTP_READ_HEADER_TIMEOUT", "HTTP_WRITE_TIMEOUT", "HTTP_IDLE_TIMEOUT",
	"DATABASE_CONNECT_TIMEOUT", "DATABASE_QUERY_TIMEOUT", "DATABASE_MAX_CONN_LIFETIME",
}

var int32EnvNames = [...]string{
	"DATABASE_MIN_CONNS", "DATABASE_MAX_CONNS",
}

func LoadConfig() (*Config, error) {
	rawValues, err := readRequiredEnv(requiredEnvNames[:])
	if err != nil {
		return nil, err
	}

	durations, err := parseDurations(rawValues, durationEnvNames[:])
	if err != nil {
		return nil, err
	}

	integers, err := parseInt32s(rawValues, int32EnvNames[:])
	if err != nil {
		return nil, err
	}

	cfg := Config{
		LogLevel:        rawValues["LOG_LEVEL"],
		ShutdownTimeout: durations["SHUTDOWN_TIMEOUT"],
		IdempotencyTTL:  durations["IDEMPOTENCY_TTL"],
		Http: HttpConfig{
			Addr:              rawValues["HTTP_ADDR"],
			ReadTimeout:       durations["HTTP_READ_TIMEOUT"],
			ReadHeaderTimeout: durations["HTTP_READ_HEADER_TIMEOUT"],
			WriteTimeout:      durations["HTTP_WRITE_TIMEOUT"],
			IdleTimeout:       durations["HTTP_IDLE_TIMEOUT"],
		},
		DB: DBConfig{
			Url:             rawValues["DATABASE_URL"],
			MaxConns:        integers["DATABASE_MAX_CONNS"],
			MinConns:        integers["DATABASE_MIN_CONNS"],
			ConnectTimeout:  durations["DATABASE_CONNECT_TIMEOUT"],
			QueryTimeout:    durations["DATABASE_QUERY_TIMEOUT"],
			MaxConnLifetime: durations["DATABASE_MAX_CONN_LIFETIME"],
		},
	}

	if cfg.DB.MaxConns < cfg.DB.MinConns {
		return nil, fmt.Errorf("max count of db connections is lower than min ones")
	}

	return &cfg, nil
}

func readRequiredEnv(names []string) (map[string]string, error) {
	values := make(map[string]string, len(names))
	for _, name := range names {
		value, ok := os.LookupEnv(name)
		if !ok || value == "" {
			return nil, fmt.Errorf("environment variable %s is required", name)
		}
		values[name] = value
	}

	return values, nil
}

func parseDurations(rawValues map[string]string, names []string) (map[string]time.Duration, error) {
	values := make(map[string]time.Duration, len(names))
	for _, name := range names {
		value, err := time.ParseDuration(rawValues[name])
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		if value <= 0 {
			return nil, fmt.Errorf("%s must be positive", name)
		}
		values[name] = value
	}

	return values, nil
}

func parseInt32s(rawValues map[string]string, names []string) (map[string]int32, error) {
	values := make(map[string]int32, len(names))
	for _, name := range names {
		value, err := strconv.ParseInt(rawValues[name], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		if value <= 0 {
			return nil, fmt.Errorf("%s must be positive", name)
		}
		values[name] = int32(value)
	}

	return values, nil
}
