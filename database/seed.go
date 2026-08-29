package database

import (
	"context"
	"log"

	"FinalProjectBE/config"
	"FinalProjectBE/repository"
	"FinalProjectBE/utils"
)

func SeedUsers(ctx context.Context, userRepo *repository.UserRepository, cfg *config.Config) error {
	seeds := []struct {
		email    string
		password string
		role     string
	}{
		{cfg.SeedKasirEmail, cfg.SeedKasirPassword, "kasir"},
		{cfg.SeedPemasakEmail, cfg.SeedPemasakPassword, "koki"},
	}

	for _, s := range seeds {
		exists, err := userRepo.ExistsByEmail(ctx, s.email)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		hash, err := utils.HashPassword(s.password)
		if err != nil {
			return err
		}

		if _, err := userRepo.Create(ctx, s.email, hash, s.role); err != nil {
			return err
		}
		log.Printf("seed: akun %s (%s) dibuat", s.email, s.role)
	}

	return nil
}