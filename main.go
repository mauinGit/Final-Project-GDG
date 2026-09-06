package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	healthCtrl := controllers.NewHealthController(pool)

	r := routes.SetupRouter(authCtrl, orderCtrl, healthCtrl, hub, cfg.JWTSecret)

		srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: r,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("server berjalan di http://localhost:%s", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("gagal menjalankan server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("sinyal shutdown diterima, menutup server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown paksa: %v", err)
	}

	log.Println("server berhenti dengan rapi")
}