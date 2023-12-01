package console

import (
	"context"
	"fmt"
	"stahl/internal/config"
	"stahl/internal/domain"
)

type DummyConsoleOutput struct {
	cfg config.OutputConfig
}

func NewDummyConsoleOutput(cfg config.OutputConfig) *DummyConsoleOutput {
	return &DummyConsoleOutput{
		cfg: cfg,
	}
}

func (d *DummyConsoleOutput) PushEvent(ctx context.Context, event domain.BaseEvent, chanelName string) error {
	fmt.Println("[GOT AN EVENT TO SEND]")
	fmt.Println(event)
	fmt.Println("[FOR CHANNEL]")
	if topicName, ok := d.cfg.TableChannel[chanelName]; ok {
		chanelName = topicName
	}
	fmt.Println(chanelName)
	return nil
}
