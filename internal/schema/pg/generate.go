package pg

import (
	"context"
	"fmt"
	"stahl/internal/config"
	"stahl/internal/domain"
	"stahl/internal/utils"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

const (
	changelogTableNameTemplate         = "%s_changelog_%s"
	changelogIndexCreatedAtTemplate    = "%s_created_at_idx"
	replicationStatusTableNameTemplate = "replication_status_%s"
	procedureNameTemplate              = "%s_changelog_proc"
	triggerNameTemplate                = "%s_changelog_stahl_trigger"
	dlqTableNameTemplate               = "dead_messages_%s"
)

type SchemaGenerator struct {
	db           *sqlx.DB
	config       config.SchemaConfig
	triggerNames map[string]string
	procNames    []string
}

func NewSchemaGenerator(db *sqlx.DB, config config.SchemaConfig) *SchemaGenerator {
	return &SchemaGenerator{
		db:           db,
		config:       config,
		triggerNames: make(map[string]string),
		procNames:    make([]string, 0, len(config.TableNames)),
	}
}

func (s *SchemaGenerator) Start(ctx context.Context) error {
	s.config.CreationTime = time.Now()
	err := s.GenerateSchema(ctx)
	if err != nil {
		return err
	}
	err = s.GenerateTriggers(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (s *SchemaGenerator) GetConfig() config.SchemaConfig {
	return s.config
}

func (s *SchemaGenerator) GenerateSchema(ctx context.Context) error {
	err := WithTx(ctx, s.db, func(ctx context.Context, tx *sqlx.Tx) error {
		err := s.checkTablesExistenceTx(ctx, tx)
		if err != nil {
			return err
		}
		replStatTable, err := s.generateReplicationStatusTableAndMethodsEnumTx(ctx, tx)
		if err != nil {
			return err
		}
		domain.DlqTableName, err = s.generateDlqTableTx(ctx, tx)
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		s.config.ReplicationStatusTableName = replStatTable
		for _, tableName := range s.config.TableNames {
			replicaTable, err := s.generateChangelogTableTableTx(ctx, tableName, tx)
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

func (s *SchemaGenerator) GenerateTriggers(ctx context.Context) error {
	err := WithTx(ctx, s.db, func(ctx context.Context, tx *sqlx.Tx) error {
		err := s.generateTriggersTx(ctx, tx)
		if err != nil {
			return fmt.Errorf("s.generateTriggersTx: %w", err)
		}
		return nil
	}, nil)
	if err != nil {
		return fmt.Errorf("generate triggers err: %w", err)
	}
	return nil
}

func (s *SchemaGenerator) generateReplicationStatusTableAndMethodsEnumTx(ctx context.Context, tx *sqlx.Tx) (string, error) {
	_, err := tx.ExecContext(ctx, DropMethodEnumQuery)
	if err != nil {
		return "", fmt.Errorf("drop enum error: %w", err)
	}
	_, err = tx.ExecContext(ctx, CreateMethodEnumQuery)
	if err != nil {
		return "", fmt.Errorf("create enum error: %w", err)
	}
	metaTableTime := utils.TimestampToString(s.config.CreationTime)
	metatableName := fmt.Sprintf(replicationStatusTableNameTemplate, metaTableTime)
	_, err = tx.ExecContext(ctx, fmt.Sprintf(CreateTableMetaDataTableQuery, metatableName))
	if err != nil {
		return "", fmt.Errorf("create metadata table error: %w", err)
	}
	return metatableName, nil
}

func (s *SchemaGenerator) generateChangelogTableTableTx(ctx context.Context, tableName string, tx *sqlx.Tx) (string, error) {
	tableTime := utils.TimestampToString(s.config.CreationTime)
	replicaTableName := fmt.Sprintf(changelogTableNameTemplate, tableName, tableTime)
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
	query, args, err := squirrel.
		Insert(s.config.ReplicationStatusTableName).
		Columns("table_name").
		Values(replicaTableName).
		PlaceholderFormat(squirrel.Dollar).ToSql()
	if err != nil {
		return "", fmt.Errorf("generate sql for insert into replica status error: %w", err)
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("insert changelog tablenames error: %w", err)
	}

	return replicaTableName, nil
}

func (s *SchemaGenerator) Drop() error {
	return WithTx(context.Background(), s.db, func(ctx context.Context, tx *sqlx.Tx) error {
		for orig, table := range s.config.ChangelogTableNames {
			_, err := tx.ExecContext(ctx, fmt.Sprintf(DropTriggerQuery, s.triggerNames[orig], orig))
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, fmt.Sprintf(DropTableQuery, table))
			if err != nil {
				return err
			}
		}
		for _, proc := range s.procNames {
			_, err := tx.ExecContext(ctx, fmt.Sprintf(DropProcedureQuery, proc))
			if err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, fmt.Sprintf(DropTableQuery, s.config.ReplicationStatusTableName))
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, fmt.Sprintf(DropTableQuery, domain.DlqTableName))
		if err != nil {
			return err
		}
		return nil
	}, nil)

}

func (s *SchemaGenerator) checkTablesExistenceTx(ctx context.Context, tx *sqlx.Tx) error {
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
		return fmt.Errorf("expected %d, got %d: %w", len(s.config.TableNames), providedTablesCount, domain.ErrorTablesDoMatchWithSchema)
	}
	return nil
}

func (s *SchemaGenerator) generateTriggersTx(ctx context.Context, tx *sqlx.Tx) error {
	for original, replica := range s.config.ChangelogTableNames {
		procName := fmt.Sprintf(procedureNameTemplate, replica)
		s.procNames = append(s.procNames, procName)
		_, err := tx.ExecContext(ctx, fmt.Sprintf(CreateCommonTriggerProcedureQuery, procName, replica, replica, replica, replica))
		if err != nil {
			return fmt.Errorf("unable to create procedure for table %s: %w", replica, err)
		}
		triggerName := fmt.Sprintf(triggerNameTemplate, replica)
		s.triggerNames[original] = triggerName
		_, err = tx.ExecContext(ctx, fmt.Sprintf(CreateTriggerForReplicaTableQuery, triggerName, original, procName))
		if err != nil {
			return fmt.Errorf("unable to create trigger for table %s: %w", replica, err)
		}
	}
	return nil
}

func (s *SchemaGenerator) generateDlqTableTx(ctx context.Context, tx *sqlx.Tx) (string, error) {
	dlqTable := fmt.Sprintf(dlqTableNameTemplate, utils.TimestampToString(s.config.CreationTime))
	_, err := tx.ExecContext(ctx, fmt.Sprintf(CreateDlqTableQuery, dlqTable))
	if err != nil {
		return "", fmt.Errorf("createDlqTableQuery: %w", err)
	}
	return dlqTable, nil
}

// GetDB returns the database connection
func (s *SchemaGenerator) GetDB() *sqlx.DB {
	return s.db
}
