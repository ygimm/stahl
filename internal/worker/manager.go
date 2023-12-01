package worker

import (
	"context"
)

type IManager interface {
	Start(ctx context.Context) <-chan error
	StartProducers(ctx context.Context) error
	StartConsumers(ctx context.Context) error
}
