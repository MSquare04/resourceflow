BACKEND_DIR := backend
BIN_DIR := $(BACKEND_DIR)/bin
APP_BIN := $(BIN_DIR)/app

.PHONY: help run test tidy build fmt vet check clean

help:
	@echo "Available commands:"
	@echo "  make run    - run backend"
	@echo "  make test   - run backend tests"
	@echo "  make tidy   - run go mod tidy"
	@echo "  make build  - build backend binary"
	@echo "  make fmt    - format backend Go code"
	@echo "  make vet    - run go vet checks"
	@echo "  make check  - run fmt + vet + test"
	@echo "  make clean  - remove build artifacts"

run:
	cd $(BACKEND_DIR) && go run ./cmd/app

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
