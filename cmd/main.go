package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/agnos-assessment/agnos-backend/internal/config"
	"github.com/agnos-assessment/agnos-backend/internal/database"
	"github.com/agnos-assessment/agnos-backend/internal/handler"
	"github.com/agnos-assessment/agnos-backend/internal/repository"
	"github.com/agnos-assessment/agnos-backend/internal/service"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.Database, database.DefaultRetryConfig)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

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
