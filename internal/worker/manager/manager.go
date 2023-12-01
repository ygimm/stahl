package manager

import (
	"context"
	"stahl/internal/config"
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
	schemaCfg   config.SchemaConfig
	producerCfg config.ProducerConfig
	consumerCfg config.ConsumerConfig
}

func NewWorkerManager(
	transfer worker.ITaskTransfer,
	storage store.IStorage,
	output output.IOutput,
	sCfg config.SchemaConfig,
	pCfg config.ProducerConfig,
	cCfg config.ConsumerConfig,
) *WorkerManager {
	return &WorkerManager{
		transfer:    transfer,
		storage:     storage,
		output:      output,
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
			p := producer.New(w.transfer, w.storage, ctx, w.producerCfg, original, replica, w.schemaCfg.ReplicationStatusTableName)
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
			c := consumer.New(ctx, w.consumerCfg, w.transfer, original, replica, w.storage, w.output)
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
