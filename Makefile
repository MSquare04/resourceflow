BACKEND_DIR := backend
BIN_DIR := $(BACKEND_DIR)/bin
APP_BIN := $(BIN_DIR)/app
MIGRATIONS_DIR := migrations
MIGRATE_TOOL := github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3

-include .env

POSTGRES_HOST ?= 127.0.0.1
POSTGRES_PORT ?= 5432
POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= postgres
POSTGRES_DB ?= resourceflow
POSTGRES_SSLMODE ?= disable

DATABASE_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSLMODE)

.PHONY: help run dev-up test tidy build fmt vet check clean migrate-up migrate-down migrate-version demo-reset

help:
	@echo "Available commands:"
	@echo "  make run    - run backend"
	@echo "  make dev-up - apply migrations and run backend"
	@echo "  make test   - run backend tests"
	@echo "  make tidy   - run go mod tidy"
	@echo "  make build  - build backend binary"
	@echo "  make fmt    - format backend Go code"
	@echo "  make vet    - run go vet checks"
	@echo "  make check  - run fmt + vet + test"
	@echo "  make migrate-up      - apply all migrations"
	@echo "  make migrate-down    - rollback one migration"
	@echo "  make migrate-version - show current migration version"
	@echo "  make demo-reset      - reset local application data and seed demo data"
	@echo "  make clean  - remove build artifacts"

run:
	cd $(BACKEND_DIR) && go run ./cmd/app

dev-up: migrate-up run

test:
	cd $(BACKEND_DIR) && go test ./...

tidy:
	cd $(BACKEND_DIR) && go mod tidy

build:
	mkdir -p $(BIN_DIR)
	cd $(BACKEND_DIR) && go build -o bin/app ./cmd/app

fmt:
	cd $(BACKEND_DIR) && go fmt ./...

vet:
	cd $(BACKEND_DIR) && go vet ./...

check: fmt vet test

clean:
	rm -rf $(BIN_DIR)

migrate-up:
	cd $(BACKEND_DIR) && go run -tags "postgres" $(MIGRATE_TOOL) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-down:
	cd $(BACKEND_DIR) && go run -tags "postgres" $(MIGRATE_TOOL) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

migrate-version:
	cd $(BACKEND_DIR) && go run -tags "postgres" $(MIGRATE_TOOL) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" version

demo-reset:
	@set -a; . ./.env.demo.local; set +a; cd backend && go run ./cmd/demo-seed
