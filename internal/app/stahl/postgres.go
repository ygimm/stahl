package stahl

import (
	"context"
	"fmt"
	"stahl/internal/config"
	pgx "stahl/internal/db/pg"

	"github.com/jmoiron/sqlx"
)

func BootstrapPostgres(ctx context.Context, dbCfg config.DatabaseConfig) (*sqlx.DB, error) {
	pgconn, err := pgx.GetPostgresConnector(ctx, domainCfgToPostgres(dbCfg))
	if err != nil {
		return nil, fmt.Errorf("GetPostgresConnector: %w", err)
	}
	db := pgx.GetSqlxConnectorPgxDriver(pgconn)
	if db == nil {
		return nil, fmt.Errorf("GetSqlxConnectorPgxDriver: %w", err)
	}
	return db, nil
}

func domainCfgToPostgres(db config.DatabaseConfig) *pgx.PostgresConfig {
	return &pgx.PostgresConfig{
		Host:        db.Host,
		Port:        db.Port,
		Database:    db.Database,
		User:        db.User,
		Password:    db.Password,
		PingPeriod:  db.PingPeriod,
		PingTimeout: db.PingTimeout,
	}
}
