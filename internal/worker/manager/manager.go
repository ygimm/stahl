package manager

import (
	"context"
	"stahl/internal/config"
	"stahl/internal/metrics"
	"stahl/internal/output"
	"stahl/internal/store"
	"stahl/internal/worker"
	"stahl/internal/worker/consumer"
	"stahl/internal/worker/producer"
)

type WorkerManager struct {
	transfer    worker.ITaskTransfer
	errChan     chan error
	storage     store.IStorage
	output      output.IOutput
	metrics     metrics.IMetrics
	schemaCfg   config.SchemaConfig
	producerCfg config.ProducerConfig
	consumerCfg config.ConsumerConfig
}

func NewWorkerManager(
	transfer worker.ITaskTransfer,
	storage store.IStorage,
	output output.IOutput,
	metrics metrics.IMetrics,
	sCfg config.SchemaConfig,
	pCfg config.ProducerConfig,
	cCfg config.ConsumerConfig,
) *WorkerManager {
	return &WorkerManager{
		transfer:    transfer,
		storage:     storage,
		output:      output,
		metrics:     metrics,
		schemaCfg:   sCfg,
		producerCfg: pCfg,
		consumerCfg: cCfg,
	}
}

func (w *WorkerManager) Start(
	ctx context.Context) <-chan error {
	w.errChan = make(chan error)
	go func() {
		err := w.StartProducers(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				w.errChan <- err
			}
		}
	}()
	go func() {
		err := w.StartConsumers(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				w.errChan <- err
			}
		}
	}()
	return w.errChan
}

func (w *WorkerManager) StartProducers(ctx context.Context) error {
	for original, replica := range w.schemaCfg.ChangelogTableNames {
		go func(original, replica string) {
			pDeps := producer.Deps{
				Storage:         w.storage,
				Transfer:        w.transfer,
				Cfg:             w.producerCfg,
				TaskChannelName: original,
				TableName:       replica,
				StatusTable:     w.schemaCfg.ReplicationStatusTableName,
			}
			p := producer.New(ctx, pDeps)
			err := p.Run()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					w.errChan <- err
				}
			}
		}(original, replica)
	}
	return nil
}

func (w *WorkerManager) StartConsumers(ctx context.Context) error {
	for original, replica := range w.schemaCfg.ChangelogTableNames {
		go func(original, replica string) {
			cDeps := consumer.Deps{
				Storage:      w.storage,
				OutSrv:       w.output,
				TransferSrv:  w.transfer,
				MetricsSrv:   w.metrics,
				Cfg:          w.consumerCfg,
				OrigTable:    original,
				ReplicaTable: replica,
			}
			c := consumer.New(ctx, cDeps)
			err := c.Run()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					w.errChan <- err
				}
			}
		}(original, replica)
	}
	return nil
}
