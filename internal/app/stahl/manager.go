package stahl

import (
	"context"
	"fmt"
	"log"
	"stahl/internal/config"
	"stahl/internal/domain"
	"stahl/internal/output"
	"stahl/internal/output/console"
	"stahl/internal/output/kafka"
	"stahl/internal/store"
	"stahl/internal/store/storepg"
	"stahl/internal/worker"
	"stahl/internal/worker/manager"
	"stahl/internal/worker/transfer"

	"github.com/Shopify/sarama"
)

func GetManager(cfg config.Summary) (worker.IManager, error) {
	s, err := getStorage(cfg.Drivers.Db)
	if err != nil {
		return nil, err
	}
	o, err := getOutput(cfg.Drivers.Output)
	if err != nil {
		return nil, err
	}
	t := transfer.NewTaskTransfer(cfg.Schema.TableNames)
	m := manager.NewWorkerManager(t, s, o, cfg.Schema, cfg.Producer, cfg.Consumer)
	return m, nil
}

func getStorage(cfg config.DatabaseConfig) (store.IStorage, error) {
	dbType, ok := domain.DriverNameToType[cfg.DriverName]
	if !ok {
		return nil, domain.ErrorUnknownDriverName
	}
	var s store.IStorage
	switch dbType {
	case domain.Postgres:
		db, err := BootstrapPostgres(context.Background(), cfg)
		if err != nil {
			return nil, err
		}
		s = storepg.NewStorage(db)
	default:
		return nil, domain.ErrorUnknownDriverName
	}
	return s, nil
}

func getOutput(cfg config.OutputConfig) (output.IOutput, error) {
	outType := domain.OutDriverNameToType[cfg.DriverName]
	var o output.IOutput
	switch outType {
	case domain.Kafka:
		producerCfg := sarama.NewConfig()
		producerCfg.Producer.Return.Successes = true
		producer, err := sarama.NewSyncProducer(cfg.Brokers, producerCfg)
		if err != nil {
			return nil, fmt.Errorf("sarama.NewSyncProducer: %w", err)
		}
		o = kafka.NewOutput(producer, cfg)
	default:
		log.Default().Println("[CONFIGURATION OF DUMMY CONSOLE OUTPUT]")
		o = console.NewDummyConsoleOutput(cfg)
	}

	return o, nil
}
