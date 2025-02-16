FROM golang:1.24.0-alpine AS builder

WORKDIR /app

# Install git for fetching dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build arguments for version information
ARG VERSION=development
ARG COMMIT=unknown

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X main.Version=${VERSION} -X main.Commit=${COMMIT}" -o ghactions-updater ./cmd/ghactions-updater

# Create final minimal image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/ghactions-updater .

# Set the entrypoint
ENTRYPOINT ["/app/ghactions-updater"]
