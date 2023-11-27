package pg

import (
	"context"
	"fmt"
	"stahl/internal/config"
	"stahl/internal/schema"
	"stahl/internal/utils"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

const (
	changelogTableNameTemplate         = "%s_changelog_%s"
	changelogIndexCreatedAtTemplate    = "%s_created_at_idx"
	replicationStatusTableNameTemplate = "replication_status_%s"
)

type SchemaGenerator struct {
	db     *sqlx.DB
	config config.SchemaConfig
}

func NewSchemaGenerator(db *sqlx.DB, config config.SchemaConfig) *SchemaGenerator {
	return &SchemaGenerator{db: db, config: config}
}
func (s *SchemaGenerator) GetConfig() config.SchemaConfig {
	return s.config
}

func (s *SchemaGenerator) Generate(ctx context.Context) error {
	err := WithTx(ctx, s.db, func(ctx context.Context, tx *sqlx.Tx) error {
		err := s.checkTablesExistance(ctx, tx)
		if err != nil {
			return err
		}
		replStatTable, err := s.generateReplicationStatusTableAndMethodsEnum(ctx, tx)
		if err != nil {
			return err
		}
		s.config.ReplicationStatusTableName = replStatTable
		for _, tableName := range s.config.TableNames {
			replicaTable, err := s.generateChangelogTableTable(ctx, tableName, tx)
			if err != nil {
				return err
			}
			s.config.ChangelogTableNames[tableName] = replicaTable
		}
		return nil
	}, nil)

	if err != nil {
		return fmt.Errorf("generate err: %w", err)
	}
	return nil
}

func (s *SchemaGenerator) generateReplicationStatusTableAndMethodsEnum(ctx context.Context, tx *sqlx.Tx) (string, error) {
	_, err := tx.ExecContext(ctx, DropMethodEnumQuery)
	if err != nil {
		return "", fmt.Errorf("drop enum error: %w", err)
	}
	_, err = tx.ExecContext(ctx, CreateMethodEnumQuery)
	if err != nil {
		return "", fmt.Errorf("create enum error: %w", err)
	}
	metaTableUUID := utils.NewUuidWoDashes()
	metatableName := fmt.Sprintf(replicationStatusTableNameTemplate, metaTableUUID)
	_, err = tx.ExecContext(ctx, fmt.Sprintf(CreateTableMetaDataTableQuery, metatableName))
	if err != nil {
		return "", fmt.Errorf("create metadata table error: %w", err)
	}
	return metatableName, nil
}

func (s *SchemaGenerator) generateChangelogTableTable(ctx context.Context, tableName string, tx *sqlx.Tx) (string, error) {
	tableUUID := utils.NewUuidWoDashes()
	replicaTableName := fmt.Sprintf(changelogTableNameTemplate, tableName, tableUUID)
	_, err := tx.ExecContext(ctx, fmt.Sprintf(CreateChangelogTableQuery, replicaTableName))
	if err != nil {
		return "", fmt.Errorf("create table error: %w", err)
	}
	indexName := fmt.Sprintf(changelogIndexCreatedAtTemplate, replicaTableName)
	_, err = tx.ExecContext(ctx, fmt.Sprintf(CreateCreatedAtIdxForChangelogTableQuery,
		indexName, replicaTableName))
	if err != nil {
		return "", fmt.Errorf("create index error: %w", err)
	}
	return replicaTableName, nil
}

func (s *SchemaGenerator) checkTablesExistance(ctx context.Context, tx *sqlx.Tx) error {
	query, args, err :=
		squirrel.Select("count(*)").
			From("pg_tables").
			Where(squirrel.Eq{
				"schemaname": "public"},
			).
			Where(squirrel.Eq{
				"tablename": s.config.TableNames,
			}).PlaceholderFormat(squirrel.Dollar).
			ToSql()
	if err != nil {
		return err
	}
	var providedTablesCount int
	err = tx.GetContext(ctx, &providedTablesCount, query, args...)
	if err != nil {
		return fmt.Errorf("cannot select table names: %w", err)
	}
	if providedTablesCount != len(s.config.TableNames) {
		return fmt.Errorf("expected %d, got %d: %w", len(s.config.TableNames), providedTablesCount, schema.ErrorTablesDoMatchWithSchema)
	}
	return nil
}
