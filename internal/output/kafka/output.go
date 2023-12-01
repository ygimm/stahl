package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"stahl/internal/config"
	"stahl/internal/domain"

	"github.com/Shopify/sarama"
	"github.com/google/uuid"
)

type Output struct {
	producer sarama.SyncProducer
	cfg      config.OutputConfig
}

func NewOutput(producer sarama.SyncProducer, cfg config.OutputConfig) *Output {
	return &Output{
		producer: producer,
		cfg:      cfg,
	}
}

func (o *Output) PushEvent(ctx context.Context, event domain.BaseEvent, chanelName string) error {
	bytes, err := json.Marshal(&event)
	if err != nil {
		return fmt.Errorf("json.Marshall: %w", err)
	}
	enc := sarama.ByteEncoder(bytes)
	key := sarama.StringEncoder(uuid.New().String())
	if topicName, ok := o.cfg.TableChannel[chanelName]; ok {
		chanelName = topicName
	}
	_, _, err = o.producer.SendMessage(&sarama.ProducerMessage{
		Topic: chanelName,
		Key:   key,
		Value: enc,
	})
	if err != nil {
		return fmt.Errorf("o.producer.SendMessage: %w", err)
	}
	return nil
}
