package manager

import (
	"context"
	"fmt"
	"stahl/internal/config"
	"stahl/internal/metrics"
	"stahl/internal/output"
	"stahl/internal/store"
	"stahl/internal/worker"
	"stahl/internal/worker/consumer"
	"stahl/internal/worker/producer"
	"sync"
	"time"
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

	// New fields for statistics tracking
	statsMu           sync.RWMutex
	tableProcessed    map[string]uint64
	tablePending      map[string]uint64
	lastProcessedTime map[string]time.Time

	// Field to track active producers and consumers
	producers map[string]worker.IChangesProducer
	consumers map[string]worker.IChangesConsumer
	isRunning bool
	stopMu    sync.Mutex
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
		errChan:     make(chan error),
		storage:     storage,
		output:      output,
		metrics:     metrics,
		schemaCfg:   sCfg,
		producerCfg: pCfg,
		consumerCfg: cCfg,

		// Initialize statistics tracking
		tableProcessed:    make(map[string]uint64),
		tablePending:      make(map[string]uint64),
		lastProcessedTime: make(map[string]time.Time),

		// Initialize worker tracking
		producers: make(map[string]worker.IChangesProducer),
		consumers: make(map[string]worker.IChangesConsumer),
		isRunning: false,
	}
}

func (w *WorkerManager) Start(
	ctx context.Context) <-chan error {
	w.stopMu.Lock()
	defer w.stopMu.Unlock()

	// If already running, return existing error channel
	if w.isRunning {
		return w.errChan
	}

	// Create new error channel
	w.errChan = make(chan error)

	// Mark as running
	w.isRunning = true

	fmt.Println("Starting WorkerManager...")

	// We'll let the StartProducers and StartConsumers be called separately
	// by the caller to give more control over the startup process

	// Monitor context cancellation
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
	w.stopMu.Lock()
	defer w.stopMu.Unlock()

	if !w.isRunning {
		return fmt.Errorf("manager is not running, call Start() first")
	}

	fmt.Println("Starting producers...")

	// Clear existing producers
	w.producers = make(map[string]worker.IChangesProducer)

	// Create and start a producer for each table
	for original, replica := range w.schemaCfg.ChangelogTableNames {
		pDeps := producer.Deps{
			Storage:         w.storage,
			Transfer:        w.transfer,
			Cfg:             w.producerCfg,
			TaskChannelName: original,
			TableName:       replica,
			StatusTable:     w.schemaCfg.ReplicationStatusTableName,
		}
		p := producer.New(ctx, pDeps)

		// Store producer reference
		w.producers[original] = p

		// Initialize stats for this table
		w.updateTableStats(original, 0, 0)

		// Start producer in a goroutine
		go func(p worker.IChangesProducer, tableName string) {
			err := p.Run()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					fmt.Printf("Producer error for table %s: %v\n", tableName, err)
					w.errChan <- fmt.Errorf("producer for table %s: %w", tableName, err)
				}
			}
		}(p, original)
	}

	fmt.Printf("Started %d producers\n", len(w.producers))
	return nil
}

func (w *WorkerManager) StartConsumers(ctx context.Context) error {
	w.stopMu.Lock()
	defer w.stopMu.Unlock()

	if !w.isRunning {
		return fmt.Errorf("manager is not running, call Start() first")
	}

	fmt.Println("Starting consumers...")

	// Clear existing consumers
	w.consumers = make(map[string]worker.IChangesConsumer)

	// Create and start a consumer for each table
	for original, replica := range w.schemaCfg.ChangelogTableNames {
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

		// Store consumer reference
		w.consumers[original] = c

		// Start consumer in a goroutine
		go func(c worker.IChangesConsumer, tableName string) {
			err := c.Run()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					fmt.Printf("Consumer error for table %s: %v\n", tableName, err)
					w.errChan <- fmt.Errorf("consumer for table %s: %w", tableName, err)
				}
			}
		}(c, original)
	}

	fmt.Printf("Started %d consumers\n", len(w.consumers))
	return nil
}

// GetTableStats returns the number of processed and pending events for a table
func (w *WorkerManager) GetTableStats(ctx context.Context, tableName string) (uint64, uint64) {
	w.statsMu.RLock()
	defer w.statsMu.RUnlock()

	processed := w.tableProcessed[tableName]
	pending := w.tablePending[tableName]

	return processed, pending
}

// GetLastProcessedTime returns the timestamp of the last processed event for a table
func (w *WorkerManager) GetLastProcessedTime(ctx context.Context, tableName string) time.Time {
	w.statsMu.RLock()
	defer w.statsMu.RUnlock()

	lastTime, exists := w.lastProcessedTime[tableName]
	if !exists {
		return time.Time{} // Return zero time if no events have been processed
	}

	return lastTime
}

// updateTableStats updates the statistics for a table
func (w *WorkerManager) updateTableStats(tableName string, processedCount uint64, pendingCount uint64) {
	w.statsMu.Lock()
	defer w.statsMu.Unlock()

	w.tableProcessed[tableName] = processedCount
	w.tablePending[tableName] = pendingCount
	w.lastProcessedTime[tableName] = time.Now()
}

// Stop stops all producers and consumers
func (w *WorkerManager) Stop() {
	w.stopMu.Lock()
	defer w.stopMu.Unlock()

	if !w.isRunning {
		return
	}

	// Inform that we're stopping
	fmt.Println("Stopping WorkerManager...")

	// Close error channel to signal shutdown
	close(w.errChan)

	// Attempt to flush any pending output
	// This is important to ensure all messages are delivered before shutdown
	if flusher, ok := w.output.(interface{ Flush() error }); ok {
		if err := flusher.Flush(); err != nil {
			fmt.Printf("Error flushing output: %v\n", err)
		}
	}

	// Attempt to close output if it implements Closer
	if closer, ok := w.output.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			fmt.Printf("Error closing output: %v\n", err)
		}
	}

	// We're not running anymore
	w.isRunning = false

	// Clear references to producers and consumers
	// This will help with garbage collection
	w.producers = make(map[string]worker.IChangesProducer)
	w.consumers = make(map[string]worker.IChangesConsumer)

	fmt.Println("WorkerManager stopped successfully")
}
