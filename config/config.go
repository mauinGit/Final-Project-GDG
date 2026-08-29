package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUser     string
	DBPassword string
	DBName     string
	DBHost     string
	DBPort     string
	AppPort    string
	JWTSecret  string

	SeedKasirEmail      string
	SeedKasirPassword   string
	SeedPemasakEmail    string
	SeedPemasakPassword string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		AppPort:    os.Getenv("APP_PORT"),
		JWTSecret:  os.Getenv("JWT_SECRET"),

		SeedKasirEmail:      os.Getenv("SEED_KASIR_EMAIL"),
		SeedKasirPassword:   os.Getenv("SEED_KASIR_PASSWORD"),
		SeedPemasakEmail:    os.Getenv("SEED_PEMASAK_EMAIL"),
		SeedPemasakPassword: os.Getenv("SEED_PEMASAK_PASSWORD"),
	}

	if cfg.DBUser == "" || cfg.DBPassword == "" || cfg.DBName == "" ||
		cfg.DBHost == "" || cfg.DBPort == "" {
		return nil, fmt.Errorf("konfigurasi database tidak lengkap, cek file .env")
	}

	return cfg, nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}