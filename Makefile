.PHONY: all backend frontend dev db clean skaffold-dev skaffold-stop

all: dev

# Start PostgreSQL
db:
	docker run -d --name mcp-builder-db \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_DB=mcp_builder \
		-p 5432:5432 \
		postgres:16-alpine

# Stop PostgreSQL
db-stop:
	docker stop mcp-builder-db && docker rm mcp-builder-db

# Install backend dependencies
backend-deps:
	cd backend && go mod download

# Run backend
backend:
	cd backend && go run ./cmd/server

# Install frontend dependencies
frontend-deps:
	cd frontend && npm install

# Run frontend
frontend:
	cd frontend && npm run dev

# Run both in development mode (requires tmux or separate terminals)
dev:
	@echo "Run the following commands in separate terminals:"
	@echo "  make db        # Start PostgreSQL (first time)"
	@echo "  make backend   # Start Go backend"
	@echo "  make frontend  # Start React frontend"

# Build backend
build-backend:
	cd backend && go build -o bin/server ./cmd/server

# Build frontend
build-frontend:
	cd frontend && npm run build

# Build all
build: build-backend build-frontend

# Run full stack on local Kubernetes (keeps namespace/PVC on exit)
skaffold-dev:
	./scripts/skaffold-dev.sh

# Scale app + postgres to 0 in make-mcp (keeps namespace and PVCs)
skaffold-stop:
	./scripts/skaffold-stop.sh

# Clean up
clean:
	rm -rf backend/bin
	rm -rf frontend/dist
	rm -rf frontend/node_modules

# Run tests
test:
	cd backend && go test ./...

# Lint
lint:
	cd backend && go vet ./...
