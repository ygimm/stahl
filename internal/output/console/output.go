package console

import (
	"context"
	"fmt"
	"stahl/internal/domain"
)

type DummyConsoleOutput struct {
}

func NewDummyConsoleOutput() *DummyConsoleOutput {
	return &DummyConsoleOutput{}
}

func (d *DummyConsoleOutput) PushEvent(ctx context.Context, event domain.BaseEvent, chanelName string) error {
	fmt.Println("[GOT AN EVENT TO SEND]")
	fmt.Println(event)
	fmt.Println("[FOR CHANNEL]")
	fmt.Println(chanelName)
	return nil
}
