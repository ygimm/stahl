package worker

import (
	"context"
	"stahl/internal/domain"
)

type ITaskOutputTransfer interface {
	GetTaskChan(taskQueueName string) <-chan domain.Task
}

type ITaskInputTransfer interface {
	ScheduleSendTask(ctx context.Context, taskQueueName string, task domain.Task) error
}

type ITaskTransfer interface {
	ITaskInputTransfer
	ITaskOutputTransfer
}
