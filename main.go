package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
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
	"FinalProjectBE/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("gagal memuat konfigurasi: %v", err)
	}
	appLog := logger.New(cfg.AppEnv)
	slog.SetDefault(appLog)

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

	r := routes.SetupRouter(authCtrl, orderCtrl, healthCtrl, hub, cfg.JWTSecret, appLog, cfg.AllowedOrigins())

	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: r,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		appLog.Info("server berjalan", "port", cfg.AppPort, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("gagal menjalankan server: %v", err)
		}
	}()

	<-ctx.Done()
	appLog.Info("sinyal shutdown diterima, menutup server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		appLog.Error("shutdown paksa", "error", err)
	}

	appLog.Info("server berhenti dengan rapi")
}