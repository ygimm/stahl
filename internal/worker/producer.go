package worker

import (
	"context"
	"stahl/internal/domain"
)

type IChangesProducer interface {
	Run() error
	GetMessages(ctx context.Context) ([]int64, error)
	PushSingleTask(ctx context.Context, task domain.Task) error
	PushTasks(ctx context.Context, tasks []domain.Task) error
}
