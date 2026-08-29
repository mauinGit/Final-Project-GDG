package main

import (
	"context"
	"fmt"
	"log"

	"FinalProjectBE/config"
	"FinalProjectBE/database"
	"FinalProjectBE/repository"
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
	if err := database.SeedUsers(context.Background(), userRepo, cfg); err != nil {
		log.Fatalf("gagal seeding user: %v", err)
	}

	fmt.Println("OrderFlow API — database & seeding siap ✅")
}