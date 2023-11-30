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
	transfer worker.ITaskTransfer
	errChan  chan error
	storage  store.IStorage
	output   output.IOutput
}

func NewWorkerManager(transfer worker.ITaskTransfer, storage store.IStorage, output output.IOutput) *WorkerManager {
	return &WorkerManager{transfer: transfer, storage: storage, output: output}
}

func (w *WorkerManager) Start(
	ctx context.Context,
	sCfg config.SchemaConfig,
	pCfg config.ProducerConfig,
	cCfg config.ConsumerConfig) <-chan error {
	w.errChan = make(chan error)
	go func() {
		err := w.StartProducers(ctx, sCfg, pCfg)
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
		err := w.StartConsumers(ctx, sCfg, cCfg)
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

func (w *WorkerManager) StartProducers(ctx context.Context, config config.SchemaConfig, wCfg config.ProducerConfig) error {
	for original, replica := range config.ChangelogTableNames {
		go func(original, replica string) {
			p := producer.New(w.transfer, w.storage, ctx, wCfg, original, replica, config.ReplicationStatusTableName)
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

func (w *WorkerManager) StartConsumers(ctx context.Context, sCfg config.SchemaConfig, cCfg config.ConsumerConfig) error {
	for original, replica := range sCfg.ChangelogTableNames {
		go func(original, replica string) {
			c := consumer.New(ctx, cCfg, w.transfer, original, replica, w.storage, w.output)
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
