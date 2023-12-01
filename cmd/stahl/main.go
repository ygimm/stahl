package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"stahl/internal/app/stahl"
	"stahl/internal/config"
	"syscall"
)

func main() {
	ctx := context.Background()

	cfg, err := config.ParseConfig()
	if err != nil {
		log.Fatal(err)
	}

	generator, err := stahl.GetSchemaGeneratorByDriverName(ctx, "pg", cfg.UserSchema)
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

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		cancel()
		fmt.Println("shutdown with an error: ", err)
	case <-exit:
		cancel()
		fmt.Println("shutdown by sigterm")
	}

}
