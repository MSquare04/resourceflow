package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"

	"resourceflow/backend/internal/auth"
	"resourceflow/backend/internal/config"
	"resourceflow/backend/internal/db"
	"resourceflow/backend/internal/demo"
)

func main() {
	_ = godotenv.Load(".env", "../.env")

	cfg := config.Load()
	if err := demo.ValidateEnvironment(cfg.Env); err != nil {
		exitWithError(err)
	}
	if err := demo.ValidateConfirmation(); err != nil {
		exitWithError(err)
	}

	password, err := demo.SeedPasswordFromEnv()
	if err != nil {
		exitWithError(err)
	}

	postgres, err := db.NewPostgres(cfg.Postgres)
	if err != nil {
		exitWithError(fmt.Errorf("init postgres: %w", err))
	}
	defer postgres.Close()

	if err := postgres.Ping(); err != nil {
		exitWithError(fmt.Errorf("ping postgres: %w", err))
	}

	resetter := demo.NewResetter(postgres, auth.NewBcryptHasher(), time.Now().UTC())
	summary, err := resetter.ResetAndSeed(context.Background(), cfg.Env, cfg.Postgres.DBName, password)
	if err != nil {
		exitWithError(err)
	}

	printSummary(summary)
}

func printSummary(summary demo.Summary) {
	fmt.Printf("Demo reset completed successfully.\n")
	fmt.Printf("Environment: %s\n", summary.Environment)
	fmt.Printf("Database: %s\n", summary.DatabaseName)
	fmt.Printf("Departments: %d\n", summary.Counts.Departments)
	fmt.Printf("Users: %d\n", summary.Counts.Users)
	fmt.Printf("Resource categories: %d\n", summary.Counts.Categories)
	fmt.Printf("Resource types: %d\n", summary.Counts.Types)
	fmt.Printf("Booking rules: %d\n", summary.Counts.Rules)
	fmt.Printf("Resources: %d\n", summary.Counts.Resources)
	fmt.Printf("Unavailability intervals: %d\n", summary.Counts.Unavailability)
	fmt.Printf("Bookings: %d\n", summary.Counts.Bookings)
	fmt.Printf("\nDemo accounts:\n")
	for _, account := range summary.Accounts {
		status := "active"
		if !account.IsActive {
			status = "inactive"
		}
		fmt.Printf("- %s | %s | %s | %s | %s\n", account.Email, account.FullName, account.Role, account.Department, status)
	}
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "demo reset failed: %v\n", err)
	os.Exit(1)
}
