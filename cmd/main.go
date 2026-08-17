package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"github.com/agnos-assessment/agnos-backend/internal/config"
	"github.com/agnos-assessment/agnos-backend/internal/handler"
	"github.com/agnos-assessment/agnos-backend/internal/repository"
	"github.com/agnos-assessment/agnos-backend/internal/service"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.Database.DSN())
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	r := gin.Default()

	healthRepo := repository.NewHealthRepository(db)
	healthSvc := service.NewHealthService(healthRepo)
	healthHandler := handler.NewHealthHandler(healthSvc)

	r.GET("/health", healthHandler.GetHealth)

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
