package consumer

import (
	"context"
	"log"
	"stahl/internal/config"
	"stahl/internal/domain"
	"stahl/internal/metrics"
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
	metrics  metrics.IMetrics

	cfg            config.ConsumerConfig
	inChannelName  string
	outChannelName string
	tableName      string

	glCtx context.Context
	tasks []domain.Task
	timer *time.Timer
}

type Deps struct {
	Storage     store.IConsumerStorage
	OutSrv      output.IOutput
	TransferSrv worker.ITaskOutputTransfer
	MetricsSrv  metrics.IMetrics

	Cfg          config.ConsumerConfig
	OrigTable    string
	ReplicaTable string
}

func New(ctx context.Context, deps Deps) worker.IChangesConsumer {
	if deps.Cfg.MaxBatchWait == 0 {
		deps.Cfg.MaxBatchWait = defaultMaxWaitTime
	}
	if deps.Cfg.MaxTasksBatch == 0 {
		deps.Cfg.MaxTasksBatch = defaultMaxTasksBatch
	}
	return &ChangesConsumer{
		storage:  deps.Storage,
		outSrv:   deps.OutSrv,
		transfer: deps.TransferSrv,
		metrics:  deps.MetricsSrv,

		inChannelName:  deps.OrigTable,
		outChannelName: deps.OrigTable,
		tableName:      deps.ReplicaTable,
		cfg:            deps.Cfg,

		glCtx: ctx,
		tasks: make([]domain.Task, 0, deps.Cfg.MaxTasksBatch),
		timer: time.NewTimer(deps.Cfg.MaxBatchWait),
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
