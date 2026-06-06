FRONTEND_DIR = ./web
BACKEND_DIR = .

.PHONY: all build-frontend build-backend start-backend test-backend

all: build-frontend start-backend

build-frontend:
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && bun install && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

build-backend:
	@echo "Building backend..."
	@cd $(BACKEND_DIR) && go build -ldflags "-X 'github.com/zhongruan0522/new-api/common.Version=$(shell git rev-parse HEAD)'" -o new-api

test-backend:
	@echo "Testing backend..."
	@cd $(BACKEND_DIR) && ./scripts/go-test-backend.sh

start-backend:
	@echo "Starting backend dev server..."
	@cd $(BACKEND_DIR) && go run main.go &
