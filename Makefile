.PHONY: dev build test lint clean frontend docker ci check

# Build frontend and copy to embed dir
frontend:
	cd frontend && npm ci && npm run build
	rm -rf internal/frontend/dist
	cp -r frontend/dist internal/frontend/dist

# Build Go binary (requires frontend to be built first)
build: frontend
	GOWORK=off CGO_ENABLED=0 go build -o fishscale ./cmd/fishscale

# Run in dev mode
dev:
	FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./fish.db FISHSCALE_PHOTO_DIR=./photos GOWORK=off go run ./cmd/fishscale

# Run all tests
test:
	GOWORK=off go test ./... -v -race
	cd frontend && npm test

# Run linters
lint:
	GOWORK=off golangci-lint run ./...
	cd frontend && npm run lint

# Clean build artifacts
clean:
	rm -f fishscale
	rm -rf internal/frontend/dist
	rm -rf frontend/dist

# Docker build
docker:
	docker build -t fishscale:latest .

# Run all CI checks locally
ci: test lint
	@echo "Running frontend format check..."
	cd frontend && npm run format:check
	@echo "Running frontend type check..."
	cd frontend && npm run check
	@echo "Building full binary..."
	$(MAKE) build
	@echo "✓ All CI checks passed!"

# Alias for ci
check: ci
