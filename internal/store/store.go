package store

import (
	"context"
	"stahl/internal/domain"
)

type IStorage interface {
	IProducerStorage
	IConsumerStorage
}

type IProducerStorage interface {
	IDeadLetterQueue
	GetEventsIdsForTable(ctx context.Context, table, statusTable string) ([]int64, error)
}

type IConsumerStorage interface {
	IDeadLetterQueue
	GetEventsByTasks(ctx context.Context, tasks []domain.Task, table string) ([]domain.BaseEvent, error)
}

type IDeadLetterQueue interface {
	PushEventDlq(ctx context.Context, event domain.BaseEvent, originError error, table string) error
}
