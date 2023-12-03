package stahl

import (
	"context"
	"fmt"
	"stahl/internal/config"
	"stahl/internal/domain"
	"stahl/internal/schema"
	"stahl/internal/schema/pg"

	"github.com/jmoiron/sqlx"
)

func GetSchemaGeneratorByDriverName(ctx context.Context, name string, cfg config.Summary) (schema.ISchemaGenerator, error) {
	if len(cfg.UserSchema.TableNames) == 0 {
		return nil, domain.ErrorNoTablesSpecified
	}
	driver, ok := domain.DriverNameToType[name]
	if !ok {
		return nil, domain.ErrorUnknownDriverName
	}
	switch driver {
	case domain.Postgres:
		db, err := BootstrapPostgres(ctx, cfg.Drivers.Db)
		if err != nil {
			return nil, fmt.Errorf("BootstrapPostgres: %w", err)
		}
		sCfg := config.SchemaConfig{TableNames: cfg.UserSchema.TableNames, ChangelogTableNames: make(map[string]string)}
		return bootstrapPostgresGenerator(ctx, sCfg, db)
	default:
		return nil, domain.ErrorUnknownDriverName
	}
}

func bootstrapPostgresGenerator(ctx context.Context, schemaCfg config.SchemaConfig, db *sqlx.DB) (schema.ISchemaGenerator, error) {
	return pg.NewSchemaGenerator(db, schemaCfg), nil
}
