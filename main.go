package main

import (
	"context"
	"log"

	"FinalProjectBE/config"
	"FinalProjectBE/controllers"
	"FinalProjectBE/database"
	"FinalProjectBE/repository"
	"FinalProjectBE/routes"
	"FinalProjectBE/service"
	"FinalProjectBE/ws"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("gagal memuat konfigurasi: %v", err)
	}

	pool, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer pool.Close()

	if err := database.RunMigrations(context.Background(), pool, "migrations/001_init_schema.sql"); err != nil {
		log.Fatalf("gagal migrasi: %v", err)
	}

	userRepo := repository.NewUserRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)

	if err := database.SeedUsers(context.Background(), userRepo, cfg); err != nil {
		log.Fatalf("gagal seeding user: %v", err)
	}

	hub := ws.NewHub()
	go hub.Run()

	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	orderService := service.NewOrderService(orderRepo)

	authCtrl := controllers.NewAuthController(authService)
	orderCtrl := controllers.NewOrderController(orderService, hub)

	r := routes.SetupRouter(authCtrl, orderCtrl, hub, cfg.JWTSecret)

	log.Printf("server berjalan di http://localhost:%s", cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("gagal menjalankan server: %v", err)
	}
}