package config

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUser     		string
	DBPassword 		string
	DBName     		string
	DBHost     		string
	DBPort     		string
	AppPort    		string
	AppEnv     		string
	CORSOrigins 	string
	JWTSecret  		string

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
		AppEnv:     os.Getenv("APP_ENV"),
		CORSOrigins: os.Getenv("CORS_ORIGINS"),
		JWTSecret:  os.Getenv("JWT_SECRET"),

		SeedKasirEmail:      os.Getenv("SEED_KASIR_EMAIL"),
		SeedKasirPassword:   os.Getenv("SEED_KASIR_PASSWORD"),
		SeedPemasakEmail:    os.Getenv("SEED_PEMASAK_EMAIL"),
		SeedPemasakPassword: os.Getenv("SEED_PEMASAK_PASSWORD"),
	}

	if cfg.AppPort == "" {
		cfg.AppPort = "8080"
	}

	if cfg.AppEnv == "" {
		cfg.AppEnv = "development"
	}
	
	if cfg.CORSOrigins == "" {
		cfg.CORSOrigins = "http://localhost:8080"
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"DB_USER":               c.DBUser,
		"DB_PASSWORD":           c.DBPassword,
		"DB_NAME":               c.DBName,
		"DB_HOST":               c.DBHost,
		"DB_PORT":               c.DBPort,
		"JWT_SECRET":            c.JWTSecret,
		"SEED_KASIR_EMAIL":      c.SeedKasirEmail,
		"SEED_KASIR_PASSWORD":   c.SeedKasirPassword,
		"SEED_PEMASAK_EMAIL":    c.SeedPemasakEmail,
		"SEED_PEMASAK_PASSWORD": c.SeedPemasakPassword,
	}

	var missing []string
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)

	if len(missing) > 0 {
		return fmt.Errorf(
			"environment variable wajib belum diisi: %s (cek file .env)",
			strings.Join(missing, ", "),
		)
	}

	if _, err := strconv.Atoi(c.DBPort); err != nil {
		return fmt.Errorf("DB_PORT harus berupa angka, dapat %q", c.DBPort)
	}

	if _, err := strconv.Atoi(c.AppPort); err != nil {
		return fmt.Errorf("APP_PORT harus berupa angka, dapat %q", c.AppPort)
	}

	if len(c.JWTSecret) < 32 {
		return fmt.Errorf(
			"JWT_SECRET terlalu pendek (%d karakter), minimal 32 karakter",
			len(c.JWTSecret),
		)
	}

	if c.AppEnv != "development" && c.AppEnv != "production" {
		return fmt.Errorf(
			"APP_ENV harus 'development' atau 'production', dapat %q",
			c.AppEnv,
		)
	}
	
	return nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

func (c *Config) AllowedOrigins() []string {
	parts := strings.Split(c.CORSOrigins, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}