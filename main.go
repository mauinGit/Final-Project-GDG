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

	_ "FinalProjectBE/docs"
)
// @title           OrderFlow API
// @version         1.0
// @description     REST API untuk sistem kasir dan antrean dapur.
// @description     Kasir membuat pesanan, koki memprosesnya secara realtime.

// @host      localhost:8080
// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                Masukkan token dengan format: Bearer {token}
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

	if err := database.RunMigrations(context.Background(), pool, "migrations"); err != nil {
		log.Fatalf("gagal migrasi: %v", err)
	}

	userRepo := repository.NewUserRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)
	menuRepo := repository.NewMenuRepository(pool)
	reportRepo := repository.NewReportRepository(pool)
	refreshRepo := repository.NewRefreshTokenRepository(pool)

	if err := database.SeedUsers(context.Background(), userRepo, cfg); err != nil {
		log.Fatalf("gagal seeding user: %v", err)
	}

	hub := ws.NewHub()
	go hub.Run()

	authService := service.NewAuthService(userRepo, refreshRepo, cfg.JWTSecret)
	orderService := service.NewOrderService(orderRepo)
	menuService := service.NewMenuService(menuRepo)
	reportService := service.NewReportService(reportRepo)

	authCtrl := controllers.NewAuthController(authService)
	orderCtrl := controllers.NewOrderController(orderService, hub)
	healthCtrl := controllers.NewHealthController(pool)
	menuCtrl := controllers.NewMenuController(menuService)
	reportCtrl := controllers.NewReportController(reportService)

	r := routes.SetupRouter(authCtrl, orderCtrl, healthCtrl, menuCtrl, reportCtrl, hub, cfg.JWTSecret, appLog, cfg.AllowedOrigins())

	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: r,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// Bersihkan refresh token kedaluwarsa secara berkala.
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := refreshRepo.DeleteExpired(context.Background())
				if err != nil {
					appLog.Error("gagal membersihkan refresh token", "error", err)
					continue
				}
				if n > 0 {
					appLog.Info("refresh token kedaluwarsa dibersihkan", "jumlah", n)
				}
			}
		}
	}()

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