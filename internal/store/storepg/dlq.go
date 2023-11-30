package storepg

import (
	"context"
	"encoding/json"
	"fmt"
	"stahl/internal/domain"
)

func (s *Storage) PushEventDlq(ctx context.Context, event domain.BaseEvent, originError error, table string) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshall: %w", err)
	}
	_, err = s.db.ExecContext(ctx, PushDlqQuery,
		domain.DlqTableName,
		table,
		string(data),
		originError.Error(),
	)
	if err != nil {
		return fmt.Errorf("PushDlq error: %w", err)
	}
	return nil
}
