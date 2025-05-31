# syntax=docker/dockerfile:1.4
# Build stage
FROM golang:1.24.3-alpine3.21 AS builder

# Build arguments
ARG VERSION=development
ARG COMMIT=unknown
ARG BUILD_DATE
ARG USER=goapp
ARG UID=10001

# Environment variables
ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

# Install required packages with versions pinned
# Package names sorted alphanumerically
RUN apk add --no-cache --virtual .build-deps \
    ca-certificates \
    cosign \
    git \
    && addgroup -g ${UID} ${USER} \
    && adduser -D -u ${UID} -G ${USER} ${USER} \
    && mkdir -p /go/pkg/mod /go/src \
    && chown -R ${USER}:${USER} /go /app

# Switch to non-root user for build
USER ${USER}

# Copy go.mod and go.sum first to leverage Docker cache
COPY --chown=${USER}:${USER} go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY --chown=${USER}:${USER} . .

# Build the binary with security flags
RUN cd pkg/cmd/ghactions-updater/ && \
    go build -trimpath -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT}" -o ../../../ghactions-updater

# Generate SBOM for the build stage
FROM alpine:3.21 AS sbom-generator
RUN apk add --no-cache syft
COPY --from=builder /app /app
RUN syft /app -o spdx-json=/sbom.json

# Final stage
FROM alpine:3.21

# Build arguments for final stage
ARG VERSION
ARG BUILD_DATE
ARG USER=goapp
ARG UID=10001

# Runtime environment variables
ENV APP_USER=${USER} \
    APP_UID=${UID}

# Install runtime dependencies and setup user with a single RUN command to reduce layers
# Package names sorted alphanumerically for better maintainability
RUN apk add --no-cache \
    bash \
    ca-certificates \
    tzdata \
    && addgroup -g ${UID} ${USER} \
    && adduser -D -u ${UID} -G ${USER} ${USER} \
    # Create directories with appropriate permissions
    && mkdir -p /app /github/workspace /github/env /github/path /github/file_commands \
    # Set proper ownership without excessive permissions
    && chown -R ${USER}:${USER} /app /github \
    # Set appropriate permissions: 755 for directories (rwxr-xr-x)
    && find /github -type d -exec chmod 755 {} \; \
    # Set appropriate permissions: 644 for files (rw-r--r--)
    && find /github -type f -exec chmod 644 {} \; 2>/dev/null || true

WORKDIR /app

# Copy the binary, entrypoint script, and SBOM from previous stages
COPY --from=builder --chown=${USER}:${USER} /app/ghactions-updater .
COPY --from=sbom-generator /sbom.json /app/sbom.json
COPY --chown=${USER}:${USER} entrypoint.sh /app/

# Ensure the entrypoint script is executable without excessive permissions
RUN chmod 755 /app/entrypoint.sh

# Switch to non-root user
USER ${USER}

# Note: Security capabilities like --cap-drop=ALL should be applied at runtime
# Example: docker run --cap-drop=ALL --cap-add=NET_BIND_SERVICE [image]

# Add metadata
LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.authors="wyattroersma@gmail.com" \
      org.opencontainers.image.url="https://github.com/ThreatFlux/githubWorkFlowChecker" \
      org.opencontainers.image.documentation="https://github.com/ThreatFlux/githubWorkFlowChecker" \
      org.opencontainers.image.source="https://github.com/ThreatFlux/githubWorkFlowChecker" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.vendor="ThreatFlux" \
      org.opencontainers.image.title="githubWorkFlowChecker" \
      org.opencontainers.image.description="ThreatFlux GitHub Workflow Checker Tool" \
      org.opencontainers.image.licenses="MIT" \
      com.threatflux.image.created.by="Docker" \
      com.threatflux.image.created.timestamp="${BUILD_DATE}" \
      com.threatflux.sbom.path="/app/sbom.json"

# Set the entrypoint with exec form (already using best practice)
ENTRYPOINT ["/app/entrypoint.sh"]

# Improved health check with reasonable intervals and better process checking
# Using exec form rather than shell form for better reliability
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["sh", "-c", "ps -ef | grep -v grep | grep ghactions-updater || exit 1"]
