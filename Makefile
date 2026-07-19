.PHONY: all build release cross test test-nocache test-integration test-integration-nocache test-coverage bench lint clean install docker-build-ci docker-test docker-test-integration docker-test-all docker-build docker-build-full docker-run docker-run-full

# Binary name
BINARY_NAME=pixel-mcp
ifeq ($(OS),Windows_NT)
	BINARY_NAME := $(BINARY_NAME).exe
endif

# Build variables for releases
VERSION:=$(shell git describe --tags --always --dirty || echo dev)
ifeq ($(OS),Windows_NT)
	BUILD_TIME:=$(shell powershell -NoProfile -Command "[System.DateTime]::UtcNow.ToString('yyyy-MM-dd_HH:mm:ss')" || echo unknown)
else
	BUILD_TIME:=$(shell date -u '+%Y-%m-%d_%H:%M:%S' || echo unknown)
endif
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

all: lint test build

build:
	go build -o bin/$(BINARY_NAME) ./cmd/pixel-mcp

release:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/pixel-mcp

# Cross-compile the server and launcher for all release platforms
CROSS_PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

cross:
	@mkdir -p bin
	@for platform in $(CROSS_PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		echo "Building pixel-mcp-$$os-$$arch$$ext"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build $(LDFLAGS) -o bin/pixel-mcp-$$os-$$arch$$ext ./cmd/pixel-mcp || exit 1; \
		echo "Building pixel-mcp-launcher-$$os-$$arch$$ext"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -ldflags "-s -w" -o bin/pixel-mcp-launcher-$$os-$$arch$$ext ./cmd/pixel-mcp-launcher || exit 1; \
	done

test:
	go test -v -race -cover ./...

test-nocache:
	go test -v -race -cover -count=1 ./...

test-integration:
	go test -tags=integration -v ./...

test-integration-nocache:
	go test -tags=integration -v -count=1 ./...

test-coverage:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html

bench:
	go test -tags=integration -bench=. -benchmem ./pkg/aseprite ./pkg/tools

lint:
	go vet ./...
	go fmt ./...

clean:
	go clean
ifeq ($(OS),Windows_NT)
	-@powershell -NoProfile -Command "if (Test-Path bin) { Remove-Item bin -Recurse -Force }; if (Test-Path dist) { Remove-Item dist -Recurse -Force }; if (Test-Path coverage.txt) { Remove-Item coverage.txt -Force }; if (Test-Path coverage.html) { Remove-Item coverage.html -Force }"
else
	-@rm -rf bin dist coverage.txt coverage.html
endif

install:
	go install $(LDFLAGS) ./cmd/pixel-mcp

# Development helpers
run:
	go run ./cmd/pixel-mcp

deps:
	go mod download
	go mod tidy

# Docker CI targets
DOCKER_IMAGE_CI=pixel-mcp-ci:latest

docker-build-ci:
	docker build -f Dockerfile.ci -t $(DOCKER_IMAGE_CI) .

docker-test:
	@docker run --rm -v $(PWD):/workspace $(DOCKER_IMAGE_CI) bash -c '\
		mkdir -p /root/.config/pixel-mcp && \
		printf "{\"aseprite_path\":\"/build/aseprite/build/bin/aseprite\",\"temp_dir\":\"/tmp/pixel-mcp\",\"timeout\":30,\"log_level\":\"info\"}" > /root/.config/pixel-mcp/config.json && \
		cd /workspace && \
		go test -v -race -cover ./...'

docker-test-integration:
	@docker run --rm -v $(PWD):/workspace $(DOCKER_IMAGE_CI) bash -c '\
		mkdir -p /root/.config/pixel-mcp && \
		printf "{\"aseprite_path\":\"/build/aseprite/build/bin/aseprite\",\"temp_dir\":\"/tmp/pixel-mcp\",\"timeout\":30,\"log_level\":\"info\"}" > /root/.config/pixel-mcp/config.json && \
		cd /workspace && \
		go test -tags=integration -v ./...'

docker-test-all:
	@docker run --rm -v $(PWD):/workspace $(DOCKER_IMAGE_CI) bash -c '\
		mkdir -p /root/.config/pixel-mcp && \
		printf "{\"aseprite_path\":\"/build/aseprite/build/bin/aseprite\",\"temp_dir\":\"/tmp/pixel-mcp\",\"timeout\":30,\"log_level\":\"info\"}" > /root/.config/pixel-mcp/config.json && \
		cd /workspace && \
		go test -v -race -cover ./... && \
		go test -tags=integration -v ./...'

# Docker MCP Server targets
DOCKER_IMAGE=pixel-mcp:latest
DOCKER_IMAGE_FULL=pixel-mcp:full

docker-build:
	docker build -t $(DOCKER_IMAGE) -f Dockerfile .

docker-build-full:
	docker build -t $(DOCKER_IMAGE_FULL) -f Dockerfile.with-aseprite .

docker-run:
	@echo "ERROR: Lightweight image requires Aseprite volume mount."
	@echo "Use 'make docker-run-full' for self-contained image, or:"
	@echo ""
	@echo "  docker run --rm -i \\"
	@echo "    -v /path/to/aseprite:/usr/local/bin/aseprite:ro \\"
	@echo "    $(DOCKER_IMAGE)"
	@exit 1

docker-run-full:
	docker run --rm -i $(DOCKER_IMAGE_FULL)