package transfer

import (
	"context"
	"fmt"
	"stahl/internal/domain"
)

type TaskTransfer struct {
	channels map[string]chan domain.Task
}

func NewTaskTransfer(tables []string) *TaskTransfer {
	chans := make(map[string]chan domain.Task, 10)
	for _, table := range tables {
		chans[table] = make(chan domain.Task)
	}
	return &TaskTransfer{
		channels: chans,
	}
}

func (t *TaskTransfer) ScheduleSendTask(ctx context.Context, taskQueueName string, task domain.Task) error {
	tChan, ok := t.channels[taskQueueName]
	// todo
	fmt.Println("got a task", task)
	if !ok {
		return fmt.Errorf("no channel %s: %w", taskQueueName, domain.ErrorUnknownChannel)
	}
	tChan <- task
	return nil
}

func (t *TaskTransfer) GetTaskChan(taskQueueName string) <-chan domain.Task {
	return t.channels[taskQueueName]
}
