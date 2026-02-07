# Makefile for Sentinel - Kubernetes Controller

# Variables
BINARY_NAME=sentinel
DOCKER_IMAGE=sentinel:latest
KIND_CLUSTER=homelab

# Default target - shows available commands
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make build        - Build the Go binary"
	@echo "  make docker       - Build Docker image"
	@echo "  make deploy       - Build Docker image and deploy to KIND cluster"
	@echo "  make clean        - Remove built binary"
	@echo "  make test         - Run tests"
	@echo "  make run          - Run locally (requires kubeconfig)"

# Build the Go binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME)
	@echo "✅ Binary built: ./$(BINARY_NAME)"

# Build Docker image
.PHONY: docker
docker:
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "✅ Docker image built: $(DOCKER_IMAGE)"

# Build and deploy to KIND cluster
.PHONY: deploy
deploy: docker
	@echo "Loading image into KIND cluster $(KIND_CLUSTER)..."
	kind load docker-image $(DOCKER_IMAGE) --name $(KIND_CLUSTER)
	@echo "Deploying to Kubernetes..."
	kubectl apply -f manifests/install/sentinel.yaml
	@echo "✅ Deployed to KIND cluster"
	@echo ""
	@echo "Check status with: kubectl get pods -n kube-system -l app=sentinel-controller"

# Run locally (useful for development)
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME) locally..."
	./$(BINARY_NAME) start -v=2

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	@echo "✅ Cleaned"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test ./...

# Update dependencies
.PHONY: deps
deps:
	@echo "Updating dependencies..."
	go mod tidy
	@echo "✅ Dependencies updated"
