package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"stahl/internal/api"
	"stahl/internal/app/stahl"
	"stahl/internal/config"
	"strconv"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.ParseConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize schema generator for PostgreSQL
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

	// Update configuration with generated schema information
	cfg.Schema = generator.GetConfig()

	// Create manager for CDC operations
	m, err := stahl.GetManager(cfg)
	if err != nil {
		fmt.Println("stahl.GetManager: ", err)
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errCh := m.Start(runCtx)

	// Create and start API server (gRPC on port 8080, HTTP on port 8081)
	configService := api.NewConfigurationService(cfg, m, generator)
	apiServer := api.NewAPIServer(configService, 8080, 8081)

	go func() {
		if err := apiServer.Start(runCtx); err != nil {
			log.Printf("API server error: %v", err)
			cancel()
		}
	}()

	// Start Prometheus metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		metricsAddr := ":" + strconv.Itoa(int(cfg.Service.MetricsPort))
		log.Printf("Metrics server is listening on %s", metricsAddr)
		err := http.ListenAndServe(metricsAddr, nil)
		if err != nil {
			log.Printf("Metrics server error: %v", err)
			cancel()
			return
		}
	}()

	// Wait for termination signals
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

	// Stop API server on shutdown
	apiServer.Stop()
	log.Println("[SHUTDOWN] complete")
}
