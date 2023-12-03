package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"stahl/internal/app/stahl"
	"stahl/internal/config"
	"strconv"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	ctx := context.Background()

	cfg, err := config.ParseConfig()
	if err != nil {
		log.Fatal(err)
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

	cfg.Schema = generator.GetConfig()
	m, err := stahl.GetManager(cfg)
	if err != nil {
		fmt.Println("stahl.GetManager: ", err)
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	errCh := m.Start(runCtx)

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(
			":"+strconv.Itoa(int(cfg.Service.MetricsPort)),
			nil)
		if err != nil {
			cancel()
			return
		}
	}()

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	select {
	case <-runCtx.Done():
		log.Default().Println("[SHUTDOWN] with context done")
	case err := <-errCh:
		cancel()
		log.Default().Println("[SHUTDOWN] with an error: ", err)
	case <-exit:
		cancel()
		log.Default().Println("\n[SHUTDOWN] by sigterm")
	}
}
