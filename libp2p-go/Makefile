# LibP2P Learn - Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_NAME=libp2p-node
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME).exe
BINARY_DARWIN=$(BINARY_NAME)_darwin

# Build flags
LDFLAGS=-ldflags "-s -w"
BUILD_FLAGS=-v $(LDFLAGS)

.PHONY: all build clean test test-integration test-dht test-protocols deps run help

# Default target
all: deps test build

# Build the binary
build:
	$(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_NAME) .

# Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_UNIX) .

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_DARWIN) .

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_WINDOWS) .

# Run all tests
test:
	$(GOTEST) -v -timeout=10m ./...

# Run tests with race detection
test-race:
	$(GOTEST) -v -race -timeout=15m ./...

# Run specific test suites
test-node:
	$(GOTEST) -v -timeout=5m -run TestCreateNode ./...

test-dht:
	$(GOTEST) -v -timeout=10m -run TestDHT ./...

test-protocols:
	$(GOTEST) -v -timeout=5m -run TestProtocol ./...

test-integration:
	$(GOTEST) -v -timeout=15m -run TestMultiNode -run TestRelay -run TestHolePunching ./...

test-websocket:
	$(GOTEST) -v -timeout=5m -run TestWebSocket ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out -timeout=10m ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Install dependencies
deps:
	$(GOMOD) download
	$(GOMOD) verify

# Update dependencies
deps-update:
	$(GOMOD) tidy
	$(GOGET) -u ./...

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(BINARY_DARWIN)
	rm -f $(BINARY_WINDOWS)
	rm -f coverage.out coverage.html

# Run the application
run: build
	./$(BINARY_NAME)

# Run with example configuration
run-example: build
	./$(BINARY_NAME) --port 8080 --relay --websocket

# Run with WebSocket disabled
run-no-ws: build
	./$(BINARY_NAME) --port 8080 --websocket=false

# Run node 1 (for testing)
run-node1: build
	./$(BINARY_NAME) --port 8001 --relay

# Run node 2 (for testing, connects to node1)
run-node2: build
	./$(BINARY_NAME) --port 8002 --bootstrap /ip4/127.0.0.1/tcp/8001/p2p/12D3KooW...

# Generate configuration file
config:
	@echo "Generating example configuration..."
	@cat > config.json << 'EOF'
	{
	  "listen_port": 8080,
	  "bootstrap_peers": [
	    "/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	    "/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa"
	  ],
	  "max_connections": 1000,
	  "low_water": 50,
	  "high_water": 200,
	  "enable_relay": true,
	  "enable_hole_punch": true,
	  "enable_autonat": true,
	  "enable_websocket": true,
	  "log_level": "info",
	  "log_file": "logs/libp2p-node.log"
	}
	EOF
	@echo "Configuration saved to config.json"

# Create logs directory
logs:
	mkdir -p logs

# Install development tools
dev-tools:
	$(GOGET) golang.org/x/tools/cmd/goimports@latest
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Format code
fmt:
	$(GOCMD) fmt ./...
	goimports -w .

# Lint code
lint:
	golangci-lint run

# Run quick smoke test
smoke-test: build
	@echo "Running smoke test..."
	timeout 10s ./$(BINARY_NAME) --port 9999 || true
	@echo "Smoke test completed"

# Test different transport combinations
test-transports: build
	@echo "Testing TCP only..."
	timeout 5s ./$(BINARY_NAME) --port 8001 --websocket=false || true
	@echo "Testing with WebSocket..."
	timeout 5s ./$(BINARY_NAME) --port 8002 --websocket=true || true
	@echo "Testing with relay..."
	timeout 5s ./$(BINARY_NAME) --port 8003 --relay --websocket=true || true

# Show available targets
help:
	@echo "Available targets:"
	@echo "  Build targets:"
	@echo "    build         - Build the binary"
	@echo "    build-all     - Build for all platforms"
	@echo "    smoke-test    - Quick functionality test"
	@echo ""
	@echo "  Test targets:"
	@echo "    test          - Run all tests"
	@echo "    test-race     - Run tests with race detection"
	@echo "    test-node     - Run node creation tests"
	@echo "    test-dht      - Run DHT tests"
	@echo "    test-protocols - Run protocol tests"
	@echo "    test-integration - Run integration tests"
	@echo "    test-websocket - Run WebSocket tests"
	@echo "    test-coverage - Run tests with coverage report"
	@echo ""
	@echo "  Dependency targets:"
	@echo "    deps          - Install dependencies"
	@echo "    deps-update   - Update dependencies"
	@echo ""
	@echo "  Run targets:"
	@echo "    run           - Build and run the application"
	@echo "    run-example   - Run with example configuration"
	@echo "    run-no-ws     - Run without WebSocket"
	@echo "    run-node1     - Run first test node"
	@echo "    run-node2     - Run second test node"
	@echo "    test-transports - Test different transport combinations"
	@echo ""
	@echo "  Utility targets:"
	@echo "    clean         - Clean build artifacts"
	@echo "    config        - Generate example configuration"
	@echo "    logs          - Create logs directory"
	@echo "    fmt           - Format code"
	@echo "    lint          - Lint code"
	@echo "    dev-tools     - Install development tools"
	@echo "    help          - Show this help"

# Docker targets (optional)
docker-build:
	docker build -t libp2p-node .

docker-run:
	docker run -p 8080:8080 libp2p-node --port 8080

docker-run-ws:
	docker run -p 8080:8080 libp2p-node --port 8080 --websocket

# Create Docker network for testing
docker-network:
	docker network create libp2p-test || true

# Run multiple nodes with Docker
docker-test: docker-build docker-network
	docker run -d --name node1 --network libp2p-test -p 8001:8080 libp2p-node --port 8080 --relay --websocket
	sleep 2
	docker run -d --name node2 --network libp2p-test -p 8002:8080 libp2p-node --port 8080 --websocket
	docker logs -f node1 &
	docker logs -f node2

# Clean up Docker test environment
docker-clean:
	docker stop node1 node2 || true
	docker rm node1 node2 || true
	docker network rm libp2p-test || true 