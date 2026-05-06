package db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)

	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return conn, nil
}
