VERSION ?= dev
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: run run-dev build build-all build-frontend build-binaries release test clean setup

# Production: single binary with embedded frontend
build: build-frontend build-binaries

# Build frontend only
build-frontend:
	cd web && npm ci && npm run build

# Cross-compile Go binaries (frontend must be built first)
build-binaries:
	@mkdir -p dist
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/brainproxy-darwin-arm64       ./cmd/brainproxy
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/brainproxy-darwin-amd64       ./cmd/brainproxy
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/brainproxy-linux-amd64        ./cmd/brainproxy
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/brainproxy-linux-arm64        ./cmd/brainproxy
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/brainproxy-windows-amd64.exe  ./cmd/brainproxy
	@echo ""
	@echo "Build complete. Binaries in dist/:"
	@ls -lh dist/

# Cross-compile for all platforms (includes frontend build)
build-all: build-frontend build-binaries

# Package for release (tar.gz / zip), binary inside is always named "brainproxy"
release:
	@rm -rf dist/tmp && mkdir -p dist/tmp
	cp dist/brainproxy-darwin-arm64      dist/tmp/brainproxy && cd dist && tar czf brainproxy-darwin-arm64.tar.gz  -C tmp brainproxy && rm tmp/brainproxy
	cp dist/brainproxy-darwin-amd64      dist/tmp/brainproxy && cd dist && tar czf brainproxy-darwin-amd64.tar.gz  -C tmp brainproxy && rm tmp/brainproxy
	cp dist/brainproxy-linux-amd64       dist/tmp/brainproxy && cd dist && tar czf brainproxy-linux-amd64.tar.gz   -C tmp brainproxy && rm tmp/brainproxy
	cp dist/brainproxy-linux-arm64       dist/tmp/brainproxy && cd dist && tar czf brainproxy-linux-arm64.tar.gz   -C tmp brainproxy && rm tmp/brainproxy
	cp dist/brainproxy-windows-amd64.exe dist/tmp/brainproxy.exe && cd dist && zip brainproxy-windows-amd64.zip -j tmp/brainproxy.exe && rm tmp/brainproxy.exe
	@rmdir dist/tmp
	@echo ""
	@echo "Release packages (binary inside is named 'brainproxy'):"
	@ls -lh dist/*.tar.gz dist/*.zip

# Development: Go + Vite side by side
run-dev:
	@echo "Starting Go proxy on :8080 and Vite on :3000..."
	@go run ./cmd/brainproxy & \
	cd web && npm run dev & \
	wait

# Run production binary
run: build
	./brainproxy

# Quick setup for new users
setup: build
	./brainproxy setup

test:
	go test ./... -v

clean:
	rm -f brainproxy
	rm -rf web/dist
	rm -rf dist
