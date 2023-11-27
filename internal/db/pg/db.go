package pg

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type PostgresConfig struct {
	Host        string `yaml:"host"`
	Port        uint16 `yaml:"port"`
	Database    string `yaml:"database"`
	User        string `yaml:"user"`
	Password    string `yaml:"-"`
	PingTimeout time.Duration
	PingPeriod  time.Duration
}

func GetPostgresConnector(ctx context.Context, cfg *PostgresConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("user=%s dbname=%s password=%s host=%s port=%s sslmode=disable",
		cfg.User,
		cfg.Database,
		cfg.Password,
		cfg.Host,
		strconv.FormatUint(uint64(cfg.Port), 10))

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := pingDbWithRetry(ctx, db, cfg.PingTimeout, cfg.PingPeriod); err != nil {
		return nil, fmt.Errorf("pingDbWithRetry(): %w", err)
	}

	db.SetMaxOpenConns(10)
	return db, nil
}

func GetSqlxConnector(db *sql.DB, driverName string) *sqlx.DB {
	return sqlx.NewDb(db, driverName)
}

func pingDbWithRetry(ctx context.Context, db *sql.DB, timeout, period time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := db.Ping()
	for err != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(period):
			err = db.Ping()
		}
	}
	return nil
}
