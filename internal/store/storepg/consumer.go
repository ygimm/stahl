package storepg

import (
	"context"
	"fmt"
	"stahl/internal/domain"

	"github.com/Masterminds/squirrel"
)

func (s *Storage) GetEventsByTasks(ctx context.Context, tasks []domain.Task, table string) ([]domain.BaseEvent, error) {
	query, args, err := squirrel.
		Select(eventsColumns...).
		From(table).
		Where(squirrel.Eq{
			"id": tasksToInt(tasks),
		}).PlaceholderFormat(squirrel.Dollar).ToSql()
	
	if err != nil {
		return nil, fmt.Errorf("squriel.Select: %w", err)
	}
	var res []domain.BaseEvent
	err = s.db.SelectContext(ctx, &res, query, args...)
	if err != nil {
		return nil, fmt.Errorf("selectEvents error: %w", err)
	}

	return res, nil
}

var eventsColumns = []string{
	"id",
	"method",
	"created_at",
	"data",
}

func tasksToInt(tasks []domain.Task) []int64 {
	res := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		res = append(res, task.ID)
	}
	return res
}
