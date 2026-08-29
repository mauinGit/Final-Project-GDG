package main

import (
	"fmt"
	"log"

	"FinalProjectBE/config"
	"FinalProjectBE/database"
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

	fmt.Println("OrderFlow API — koneksi database berhasil ✅")
	fmt.Println("Database:", cfg.DBName, "di", cfg.DBHost+":"+cfg.DBPort)
}