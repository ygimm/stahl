package pg

import (
	"context"
	"fmt"
	"stahl/internal/utils"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

// TableSchemaGenerator provides methods to manage the schema for a single table
type TableSchemaGenerator struct {
	db                     *sqlx.DB
	tableName              string
	replicationStatusTable string
	triggerName            string
	procedureName          string
}

// NewTableSchemaGenerator creates a new TableSchemaGenerator
func NewTableSchemaGenerator(db *sqlx.DB, tableName, replicationStatusTable string) (*TableSchemaGenerator, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	if tableName == "" {
		return nil, fmt.Errorf("table name cannot be empty")
	}

	if replicationStatusTable == "" {
		return nil, fmt.Errorf("replication status table cannot be empty")
	}

	return &TableSchemaGenerator{
		db:                     db,
		tableName:              tableName,
		replicationStatusTable: replicationStatusTable,
	}, nil
}

// CreateChangelogTable creates a changelog table for the table
func (t *TableSchemaGenerator) CreateChangelogTable(ctx context.Context) (string, error) {
	creationTime := time.Now()
	timestampStr := utils.TimestampToString(creationTime)
	changelogTableName := fmt.Sprintf(changelogTableNameTemplate, t.tableName, timestampStr)

	err := WithTx(ctx, t.db, func(ctx context.Context, tx *sqlx.Tx) error {
		// Create the changelog table
		_, err := tx.ExecContext(ctx, fmt.Sprintf(CreateChangelogTableQuery, changelogTableName))
		if err != nil {
			return fmt.Errorf("create table error: %w", err)
		}

		// Create index on created_at
		indexName := fmt.Sprintf(changelogIndexCreatedAtTemplate, changelogTableName)
		_, err = tx.ExecContext(ctx, fmt.Sprintf(CreateCreatedAtIdxForChangelogTableQuery,
			indexName, changelogTableName))
		if err != nil {
			return fmt.Errorf("create index error: %w", err)
		}

		// Insert into replication status table
		query, args, err := squirrel.
			Insert(t.replicationStatusTable).
			Columns("table_name", "last_fetch_timestamp").
			Values(changelogTableName, time.Now()).
			PlaceholderFormat(squirrel.Dollar).ToSql()
		if err != nil {
			return fmt.Errorf("generate sql for insert into replica status error: %w", err)
		}

		_, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("insert changelog tablenames error: %w", err)
		}

		return nil
	}, nil)

	if err != nil {
		return "", err
	}

	return changelogTableName, nil
}

// CreateTrigger creates a trigger and procedure for the table
func (t *TableSchemaGenerator) CreateTrigger(ctx context.Context, changelogTableName string) error {
	t.procedureName = fmt.Sprintf(procedureNameTemplate, changelogTableName)
	t.triggerName = fmt.Sprintf(triggerNameTemplate, changelogTableName)

	return WithTx(ctx, t.db, func(ctx context.Context, tx *sqlx.Tx) error {
		// Create the procedure
		_, err := tx.ExecContext(ctx, fmt.Sprintf(CreateCommonTriggerProcedureQuery,
			t.procedureName, changelogTableName, changelogTableName, changelogTableName, changelogTableName))
		if err != nil {
			return fmt.Errorf("unable to create procedure for table %s: %w", changelogTableName, err)
		}

		// Create the trigger
		_, err = tx.ExecContext(ctx, fmt.Sprintf(CreateTriggerForReplicaTableQuery,
			t.triggerName, t.tableName, t.procedureName))
		if err != nil {
			return fmt.Errorf("unable to create trigger for table %s: %w", changelogTableName, err)
		}

		return nil
	}, nil)
}

// DropTrigger drops the trigger for the table
func (t *TableSchemaGenerator) DropTrigger(ctx context.Context) error {
	if t.triggerName == "" {
		t.triggerName = fmt.Sprintf(triggerNameTemplate, t.tableName)
	}

	return WithTx(ctx, t.db, func(ctx context.Context, tx *sqlx.Tx) error {
		// Drop the trigger
		_, err := tx.ExecContext(ctx, fmt.Sprintf(DropTriggerQuery, t.triggerName, t.tableName))
		if err != nil {
			return fmt.Errorf("unable to drop trigger for table %s: %w", t.tableName, err)
		}

		// Drop the procedure if it exists
		if t.procedureName != "" {
			_, err = tx.ExecContext(ctx, fmt.Sprintf(DropProcedureQuery, t.procedureName))
			if err != nil {
				return fmt.Errorf("unable to drop procedure %s: %w", t.procedureName, err)
			}
		}

		return nil
	}, nil)
}

// DropChangelogTable drops the changelog table
func (t *TableSchemaGenerator) DropChangelogTable(ctx context.Context, changelogTableName string) error {
	return WithTx(ctx, t.db, func(ctx context.Context, tx *sqlx.Tx) error {
		// Drop the changelog table
		_, err := tx.ExecContext(ctx, fmt.Sprintf(DropTableQuery, changelogTableName))
		if err != nil {
			return fmt.Errorf("unable to drop changelog table %s: %w", changelogTableName, err)
		}

		// Remove from replication status table
		query, args, err := squirrel.
			Delete(t.replicationStatusTable).
			Where(squirrel.Eq{"table_name": changelogTableName}).
			PlaceholderFormat(squirrel.Dollar).ToSql()
		if err != nil {
			return fmt.Errorf("generate sql for delete from replica status error: %w", err)
		}

		_, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("delete from replication status error: %w", err)
		}

		return nil
	}, nil)
}
