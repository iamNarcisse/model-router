.PHONY: proto build run test seed clean

# Proto generation
proto:
	@echo "Generating proto files..."
	./scripts/generate-proto.sh

# Build
build: proto
	@echo "Building Go router..."
	cd services/router && go build -o ../../bin/router ./cmd/server
	cd services/router && go build -o ../../bin/seed ./cmd/seed

build-docker:
	docker compose -f deployments/docker-compose.yaml build

# Run locally
run-deps:
	docker compose -f deployments/docker-compose.yaml up -d qdrant embedding

run-router: build
	./bin/router --config configs/config.yaml

run: run-deps
	@sleep 3  # Wait for deps
	docker compose -f deployments/docker-compose.yaml up router

run-all:
	docker compose -f deployments/docker-compose.yaml up

# Seed routes
seed-py: run-deps
	@sleep 2
	cd scripts && python seed-routes.py

seed-go: build run-deps
	@sleep 2
	./bin/seed

seed: seed-py  # Default to Python seeder

# Test
test:
	cd services/router && go test ./...
	cd services/embedding && pytest

test-integration: run-all
	@sleep 5
	./scripts/test-route.sh

# Clean
clean:
	rm -rf bin/
	docker compose -f deployments/docker-compose.yaml down -v