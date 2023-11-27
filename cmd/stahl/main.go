package main

import (
	"context"
	"fmt"
	"log"
	"stahl/internal/config"
	"stahl/internal/db/pg"
	schema "stahl/internal/schema/pg"
	"time"
)

func main() {
	ctx := context.Background()
	pgconn, err := pg.GetPostgresConnector(ctx, &pg.PostgresConfig{
		Host:        "localhost",
		Port:        6000,
		Database:    "postgres",
		User:        "postgres",
		Password:    "postgres",
		PingTimeout: 5 * time.Second,
		PingPeriod:  1 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	db := pg.GetSqlxConnector(pgconn, "postgres")
	if db == nil {
		log.Fatal("nil db")
	}
	generator := schema.NewSchemaGenerator(db, config.SchemaConfig{
		TableNames:          []string{"bebra", "boba", "users"},
		ChangelogTableNames: make(map[string]string),
	})
	err = generator.Generate(ctx)
	if err != nil {
		log.Fatal("generator.Generate:", err)
	}
	fmt.Println(generator.GetConfig())
}
