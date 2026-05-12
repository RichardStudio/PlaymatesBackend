package main

import (
	"log"
	"playmates/components/connection-manager"
	"playmates/components/db"
	"playmates/components/entrypoint"
	"playmates/components/playmates/config"
	"playmates/components/playmates/handler"
	"playmates/components/playmates/service"
	"playmates/components/repository"
	"playmates/components/sealer"
)

func main() {
	configPath := "config/config.yaml"

	cfg, err := config.New(configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
		return
	}

	db, err := db.ConnectPostgres(cfg.DbConnStr)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
		return
	}

	repository := repository.New(db)

	connectionManager := connection_manager.New()

	sealer, err := sealer.New([]byte(cfg.SealerSecret))
	if err != nil {
		log.Fatalf("Error creating sealer: %w", err)
	}

	service := service.New(db, cfg.JwtSecret, repository, connectionManager, sealer)

	handler := handler.New(cfg, db, service)

	server := entrypoint.New(handler)

	if err := server.Listen(":8080"); err != nil {
		log.Fatal(err)
	}
}
