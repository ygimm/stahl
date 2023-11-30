package schema

import (
	"context"
	"stahl/internal/config"
)

type ISchemaGenerator interface {
	GenerateSchema(ctx context.Context) error
	GenerateTriggers(ctx context.Context) error
	Start(ctx context.Context) error
	Drop() error
	GetConfig() config.SchemaConfig
}
