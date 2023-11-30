package storepg

import (
	"context"
	"fmt"
	"stahl/internal/schema/pg"

	"github.com/jmoiron/sqlx"
)

func (s *Storage) GetEventsIdsForTable(ctx context.Context, table, statusTable string) ([]int64, error) {
	var res []int64
	err := pg.WithTx(ctx, s.db, func(ctx context.Context, tx *sqlx.Tx) error {
		rows, err := tx.QueryxContext(ctx, fmt.Sprintf(GetIdsQuery, table, statusTable), table)
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
