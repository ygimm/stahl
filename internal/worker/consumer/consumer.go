package consumer

import (
	"context"
	"log"
	"stahl/internal/config"
	"stahl/internal/domain"
	"stahl/internal/output"
	"stahl/internal/store"
	"stahl/internal/worker"
	"time"
)

const (
	defaultMaxWaitTime   = 1 * time.Second
	defaultMaxTasksBatch = 10
)

type ChangesConsumer struct {
	storage  store.IConsumerStorage
	outSrv   output.IOutput
	transfer worker.ITaskOutputTransfer

	cfg            config.ConsumerConfig
	inChannelName  string
	outChannelName string
	tableName      string

	glCtx context.Context
	tasks []domain.Task
	timer *time.Timer
}

func New(ctx context.Context, cfg config.ConsumerConfig, transfer worker.ITaskOutputTransfer, origTable, replicaTable string, storage store.IConsumerStorage, outSrv output.IOutput) worker.IChangesConsumer {
	if cfg.MaxBatchWait == 0 {
		cfg.MaxBatchWait = defaultMaxWaitTime
	}
	if cfg.MaxTasksBatch == 0 {
		cfg.MaxTasksBatch = defaultMaxTasksBatch
	}
	return &ChangesConsumer{
		storage:  storage,
		outSrv:   outSrv,
		transfer: transfer,

		inChannelName:  origTable,
		outChannelName: origTable,
		tableName:      replicaTable,
		cfg:            cfg,

		glCtx: ctx,
		tasks: make([]domain.Task, 0, cfg.MaxTasksBatch),
		timer: time.NewTimer(cfg.MaxBatchWait),
	}
}

func (c *ChangesConsumer) Run() error {
	for {
		select {
		case <-c.glCtx.Done():
			return nil
		case <-c.timer.C:
			if len(c.tasks) == 0 {
				c.timer.Reset(c.cfg.MaxBatchWait)
				continue
			}
			err := c.fetchAndPushEvents()
			if err != nil {
				if c.cfg.StopOnError {
					return err
				}
			}
		case task, ok := <-c.transfer.GetTaskChan(c.inChannelName):
			if !ok {
				return nil
			}
			err := c.ProcessTask(c.glCtx, task)
			if err != nil {
				if c.cfg.StopOnError {
					return err
				}
			}
		}
	}
}

func (c *ChangesConsumer) ProcessTask(ctx context.Context, task domain.Task) error {
	c.tasks = append(c.tasks, task)
	if len(c.tasks) < c.cfg.MaxTasksBatch {
		return nil
	}
	err := c.fetchAndPushEvents()
	if c.cfg.StopOnError {
		return err
	}
	return nil
}

func (c *ChangesConsumer) GetEvents(ctx context.Context, tasks []domain.Task) ([]domain.BaseEvent, error) {
	return c.storage.GetEventsByTasks(ctx, tasks, c.tableName)
}

func (c *ChangesConsumer) fetchAndPushEvents() error {
	events, err := c.GetEvents(c.glCtx, c.tasks)
	if err != nil {
		log.Default().Println("GetEvents: ", err)
		return err
	}
	err = c.PushEventsBatch(c.glCtx, events)
	if err != nil {
		log.Default().Println("PushEventsBatch: ", err)
		return err
	}
	c.tasks = make([]domain.Task, 0, c.cfg.MaxTasksBatch)
	c.timer.Reset(c.cfg.MaxBatchWait)
	return nil
}

func (c *ChangesConsumer) PushSingeEvent(ctx context.Context, event domain.BaseEvent) error {
	return c.outSrv.PushEvent(ctx, event, c.outChannelName)
}

func (c *ChangesConsumer) PushEventsBatch(ctx context.Context, events []domain.BaseEvent) error {
	for _, e := range events {
		err := c.PushSingeEvent(ctx, e)
		if err != nil {
			if c.cfg.StopOnError {
				return err
			}
			if c.cfg.EnableDLQ {
				dErr := c.storage.PushEventDlq(ctx, e, err, c.tableName)
				if dErr != nil {
					log.Default().Println("storage.PushEventDlq: ", dErr)
				}
			}
		}
	}
	return nil
}
