package stahl

import (
	"context"
	"fmt"
	"stahl/internal/config"
	pgx "stahl/internal/db/pg"
	"stahl/internal/domain"
	"stahl/internal/schema"
	"stahl/internal/schema/pg"
	"time"
)

func GetSchemaGeneratorByDriverName(ctx context.Context, name string, tableNames config.UserSchemaConfig) (schema.ISchemaGenerator, error) {
	if len(tableNames.TableNames) == 0 {
		return nil, domain.ErrorNoTablesSpecified
	}
	driver, ok := domain.DriverNameToType[name]
	if !ok {
		return nil, domain.ErrorUnknownDriverName
	}
	switch driver {
	case domain.Postgres:
		//todo parse cfg
		cfg := &pgx.PostgresConfig{
			Host:        "localhost",
			Port:        6000,
			Database:    "postgres",
			User:        "postgres",
			Password:    "postgres",
			PingTimeout: 5 * time.Second,
			PingPeriod:  1 * time.Second,
		}
		sCfg := config.SchemaConfig{TableNames: tableNames.TableNames, ChangelogTableNames: make(map[string]string)}
		return bootstrapPostgresGenerator(ctx, sCfg, cfg)
	default:
		return nil, domain.ErrorUnknownDriverName
	}
}

func bootstrapPostgresGenerator(ctx context.Context, schemaCfg config.SchemaConfig, dbCfg *pgx.PostgresConfig) (schema.ISchemaGenerator, error) {
	pgconn, err := pgx.GetPostgresConnector(ctx, dbCfg)
	if err != nil {
		return nil, fmt.Errorf("GetPostgresConnector: %w", err)
	}
	db := pgx.GetSqlxConnectorPgxDriver(pgconn)
	if db == nil {
		return nil, fmt.Errorf("GetSqlxConnectorPgxDriver: %w", err)
	}
	return pg.NewSchemaGenerator(db, schemaCfg), nil
}
