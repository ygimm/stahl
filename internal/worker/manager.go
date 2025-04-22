package worker

import (
	"context"
	"time"
)

type IManager interface {
	Start(ctx context.Context) <-chan error
	StartProducers(ctx context.Context) error
	StartConsumers(ctx context.Context) error
	// New methods for table statistics
	GetTableStats(ctx context.Context, tableName string) (uint64, uint64) // Returns processedEvents, pendingEvents
	GetLastProcessedTime(ctx context.Context, tableName string) time.Time
	Stop() // New method to stop the manager
}
