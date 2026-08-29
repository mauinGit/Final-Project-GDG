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

	userRepo := repository.NewUserRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)

	if err := database.SeedUsers(context.Background(), userRepo, cfg); err != nil {
		log.Fatalf("gagal seeding user: %v", err)
	}

	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	orderService := service.NewOrderService(orderRepo)

	authCtrl := controllers.NewAuthController(authService)
	orderCtrl := controllers.NewOrderController(orderService)

	r := routes.SetupRouter(authCtrl, orderCtrl, cfg.JWTSecret)

	log.Printf("server berjalan di http://localhost:%s", cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("gagal menjalankan server: %v", err)
	}
}