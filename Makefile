APP_NAME=order-service
AIR=$(HOME)/go/bin/air

.PHONY: dev run build start test tidy fmt clean docker

# Development (Live Reload)
dev:
	$(AIR)

# Run without Air
run:
	go run ./cmd/main.go

# Build binary
build:
	mkdir -p bin
	go build -o bin/$(APP_NAME) ./cmd/main.go

# Run compiled binary
start: build
	./bin/$(APP_NAME)

# Run tests
test:
	go test ./...

# Download & clean dependencies
tidy:
	go mod tidy

# Format Go code
fmt:
	go fmt ./...

# Clean generated files
clean:
	rm -rf bin
	rm -rf tmp

# Build Docker image
docker:
	docker build -t $(APP_NAME) .
