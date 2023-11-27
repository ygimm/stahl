package schema

import (
	"context"
	"errors"
	"stahl/internal/config"
)

type ISchemaGenerator interface {
	Generate(ctx context.Context) error
	GetConfig() config.SchemaConfig
}

var (
	ErrorTablesDoMatchWithSchema = errors.New("provided table names do not match with tables in actual schema")
)
