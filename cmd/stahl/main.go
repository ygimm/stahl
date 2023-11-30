package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"stahl/internal/app/stahl"
	"stahl/internal/config"
	pgx "stahl/internal/db/pg"
	"stahl/internal/output/console"
	"stahl/internal/store/storepg"
	"stahl/internal/worker/manager"
	"stahl/internal/worker/transfer"
	"syscall"
	"time"
)

func main() {
	ctx := context.Background()

	// Generate schema
	// todo parse cfg
	cfg := config.UserSchemaConfig{
		TableNames: []string{"bebra"},
	}

	generator, err := stahl.GetSchemaGeneratorByDriverName(ctx, "pg", cfg)
	if err != nil {
		log.Fatal(err)
	}
	err = generator.Start(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err = generator.Drop()
		if err != nil {
			log.Fatal("drop schema: ", err)
		}
	}()

	dbCfg := &pgx.PostgresConfig{
		Host:        "localhost",
		Port:        6000,
		Database:    "postgres",
		User:        "postgres",
		Password:    "postgres",
		PingTimeout: 5 * time.Second,
		PingPeriod:  1 * time.Second,
	}
	pgconn, err := pgx.GetPostgresConnector(ctx, dbCfg)
	if err != nil {
		log.Fatal("GetPostgresConnector: %w", err)
	}
	db := pgx.GetSqlxConnectorPgxDriver(pgconn)
	if db == nil {
		log.Fatal("GetSqlxConnectorPgxDriver: %w", err)
	}

	pCfg := config.ProducerConfig{
		Period:      5 * time.Second,
		StopOnError: true,
		EnableDLQ:   false,
	}
	cCfg := config.ConsumerConfig{
		MaxTasksBatch: 1,
		MaxBatchWait:  5 * time.Second,
		StopOnError:   true,
		EnableDLQ:     false,
	}
	storage := storepg.NewStorage(db)
	t := transfer.NewTaskTransfer(cfg.TableNames)
	o := console.NewDummyConsoleOutput()
	m := manager.NewWorkerManager(t, storage, o)
	runCtx, cancel := context.WithCancel(ctx)
	errCh := m.Start(runCtx, generator.GetConfig(), pCfg, cCfg)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	select {
	case <-runCtx.Done():
		cancel()
		fmt.Println("shutdown")
	case err := <-errCh:
		cancel()
		fmt.Println("shutdown with an error: ", err)
	case <-exit:
		cancel()
		fmt.Println("shutdown by sigterm")
	}

}
