package worker

import (
	"context"
	"stahl/internal/config"
)

type IManager interface {
	Start(ctx context.Context, config config.SchemaConfig, pCfg config.ProducerConfig, consumerConfig config.ConsumerConfig) error
	StartProducers(ctx context.Context, config config.SchemaConfig, wCfg config.ProducerConfig) error
	StartConsumers(ctx context.Context, consumerConfig config.ConsumerConfig) error
}
