

.PHONY: all build run clean test fmt vet


BINARY_NAME=amf
BINARY_PATH=./bin/$(BINARY_NAME)


GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-s -w"

all: build


build:
	@echo "Building AMF..."
	@mkdir -p bin
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_PATH) ./cmd/amf


run: build
	@echo "Running AMF..."
	$(BINARY_PATH) -config config/amfcfg.json


run-config:
	@echo "Running AMF with custom config..."
	$(BINARY_PATH) -config $(CONFIG)


clean:
	@echo "Cleaning..."
	rm -rf bin/
	$(GO) clean


test:
	@echo "Running tests..."
	$(GO) test -v ./...


test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html


fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...


vet:
	@echo "Running go vet..."
	$(GO) vet ./...


staticcheck:
	@echo "Running staticcheck..."
	staticcheck ./...


deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy


generate:
	@echo "Generating code..."
	$(GO) generate ./...


docker-build:
	@echo "Building Docker image..."
	docker build -t amf:latest .


docker-run:
	@echo "Running AMF in Docker..."
	docker run -it --rm -p 8000:8000 -p 38412:38412 amf:latest


help:
	@echo "Available targets:"
	@echo "  build          - Build the AMF binary"
	@echo "  run            - Build and run the AMF"
	@echo "  run-config     - Run with custom config (CONFIG=path/to/config.json)"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  fmt            - Format code"
	@echo "  vet            - Run go vet"
	@echo "  staticcheck    - Run staticcheck"
	@echo "  deps           - Install dependencies"
	@echo "  generate       - Generate code"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run AMF in Docker"
