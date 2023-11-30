package worker

import (
	"context"
	"stahl/internal/domain"
)

type IChangesConsumer interface {
	Run() error
	ProcessTask(ctx context.Context, task domain.Task) error
	GetEvents(ctx context.Context, tasks []domain.Task) ([]domain.BaseEvent, error)
	PushSingeEvent(ctx context.Context, event domain.BaseEvent) error
	PushEventsBatch(ctx context.Context, event []domain.BaseEvent) error
}
