package output

import (
	"context"
	"stahl/internal/domain"
)

type IOutput interface {
	PushEvent(ctx context.Context, event domain.BaseEvent, chanelName string) error
}
