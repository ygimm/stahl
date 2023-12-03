package producer

import (
	"context"
	"log"
	"stahl/internal/config"
	"stahl/internal/domain"
	"stahl/internal/store"
	"stahl/internal/worker"
	"time"
)

type ChangesProducer struct {
	transfer worker.ITaskInputTransfer
	storage  store.IProducerStorage

	glCtx context.Context
	timer *time.Timer
	cfg   config.ProducerConfig

	taskChannelName string
	tableName       string
	statusTableName string
}

type Deps struct {
	Storage         store.IProducerStorage
	Transfer        worker.ITaskInputTransfer
	Cfg             config.ProducerConfig
	TaskChannelName string
	TableName       string
	StatusTable     string
}

func New(ctx context.Context, deps Deps) worker.IChangesProducer {
	return &ChangesProducer{
		transfer:        deps.Transfer,
		storage:         deps.Storage,
		glCtx:           ctx,
		cfg:             deps.Cfg,
		taskChannelName: deps.TaskChannelName,
		tableName:       deps.TableName,
		statusTableName: deps.StatusTable,
	}
}

func (c *ChangesProducer) Run() error {
	c.timer = time.NewTimer(c.cfg.Period)
	for {
		select {
		case <-c.glCtx.Done():
			log.Default().Printf("[PRODUCER %s CTX DONE]", c.tableName)
			return nil
		case <-c.timer.C:
			ids, err := c.GetMessages(c.glCtx)
			if err != nil {
				log.Default().Printf("storage.GetEventsIdsForTable table %s: %v\n", c.tableName, err)
				if c.cfg.StopOnError {
					return err
				}
			}
			err = c.PushTasks(c.glCtx, idsToTasks(ids))
			if err != nil {
				log.Default().Printf("c.PushTasks: %v", err)
				if c.cfg.StopOnError {
					return err
				}
			}
			c.timer.Reset(c.cfg.Period)
		}
	}
}

func (c *ChangesProducer) GetMessages(ctx context.Context) ([]int64, error) {
	return c.storage.GetEventsIdsForTable(ctx, c.tableName, c.statusTableName)
}

func (c *ChangesProducer) PushSingleTask(ctx context.Context, task domain.Task) error {
	err := c.transfer.ScheduleSendTask(ctx, c.taskChannelName, task)
	if err != nil {
		log.Default().Printf("c.transfer.ScheduleSendTask id %d: %v", task.ID, err)
		if c.cfg.StopOnError {
			return err
		}
	}
	return nil
}

func (c *ChangesProducer) PushTasks(ctx context.Context, tasks []domain.Task) error {
	for _, task := range tasks {
		err := c.PushSingleTask(ctx, task)
		if err != nil {
			log.Default().Printf("c.PushSingleTask id %d: %v", task.ID, err)
			if c.cfg.StopOnError {
				return err
			}
		}
	}
	return nil
}

func idsToTasks(ids []int64) []domain.Task {
	res := make([]domain.Task, 0, len(ids))
	for _, id := range ids {
		res = append(res, domain.Task{
			ID: id,
		})
	}
	return res
}
