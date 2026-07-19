package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/drogueria/drogueria-micky-backend/internal/config"
	"github.com/drogueria/drogueria-micky-backend/internal/database"
	"github.com/drogueria/drogueria-micky-backend/internal/router"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("error conectando a la base de datos: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	r := gin.Default()
	alertService := router.Setup(r, db, cfg)

	// RNF-006: ejecutar verificacion de alertas de inventario al inicio
	// del sistema como goroutine concurrente (chequeo en frio).
	go alertService.CheckInventoryAlerts()

	log.Printf("servidor escuchando en el puerto %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("error iniciando el servidor: %v", err)
	}
}