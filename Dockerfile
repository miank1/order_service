# -----------------------------
# Builder Stage
# -----------------------------
FROM golang:1.25.3-alpine AS builder

WORKDIR /app

# Install certificates for downloading modules
RUN apk add --no-cache git ca-certificates

# Copy dependency files first (leverages Docker cache)
COPY go.mod go.sum ./

# Download Go modules
RUN go mod download

# Copy the application source
COPY . .

# Build a static Linux binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w" \
    -o app \
    ./cmd/main.go


# -----------------------------
# Runtime Stage
# -----------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/app .

EXPOSE 8084

ENTRYPOINT ["./app"]
