package storepg

import (
	"context"
	"fmt"
	"log"
	"stahl/internal/schema/pg"

	"github.com/jmoiron/sqlx"
)

func (s *Storage) GetEventsIdsForTable(ctx context.Context, table, statusTable string) ([]int64, error) {
	var res []int64
	err := pg.WithTx(ctx, s.db, func(ctx context.Context, tx *sqlx.Tx) error {
		query := fmt.Sprintf(GetIdsQuery, table, statusTable)
		log.Default().Printf("query: %s", query)
		rows, err := tx.QueryxContext(ctx, query, table)
		if err != nil {
			return fmt.Errorf("select ids error: %w", err)
		}
		for rows.Next() {
			var row int64
			err := rows.Scan(&row)
			if err != nil {
				return fmt.Errorf("rows.Scan: %w", err)
			}
			res = append(res, row)
		}
		_, err = tx.ExecContext(ctx, fmt.Sprintf(UpdateLastFetchTimestampQuery, statusTable), table)
		if err != nil {
			return fmt.Errorf("UpdateLastFetchTimestampQuery error: %w", err)
		}
		return nil
	}, nil)
	if err != nil {
		return nil, err
	}

	return res, nil
}
