# syntax=docker/dockerfile:1.4
# Build stage
FROM golang:1.24.0-alpine3.21 AS builder

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
# Update these versions based on Alpine 3.21 package versions
RUN apk add --no-cache --virtual .build-deps \
    git=2.47.2-r0 \
    ca-certificates=20241121-r1 \
    cosign=2.4.1-r2 \
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
RUN apk add --no-cache syft=1.19.0-r2
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

# Install runtime dependencies and setup user
RUN apk add --no-cache \
    ca-certificates=20241121-r1 \
    bash=5.2.37-r0 \
    tzdata=2025a-r0 \
    && addgroup -g ${UID} ${USER} \
    && adduser -D -u ${UID} -G ${USER} ${USER} \
    && mkdir -p /app \
    && chown -R ${USER}:${USER} /app

WORKDIR /app

# Copy the binary, entrypoint script, and SBOM from previous stages
COPY --from=builder --chown=${USER}:${USER} /app/ghactions-updater .
COPY --from=sbom-generator /sbom.json /app/sbom.json
COPY --chown=${USER}:${USER} entrypoint.sh /app/

# Ensure the entrypoint script is executable
RUN chmod +x /app/entrypoint.sh

# Switch to non-root user
USER ${USER}

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

# Best practice: Drop all capabilities and enforce no-new-privileges at runtime
# Use --security-opt=no-new-privileges:true --security-opt=seccomp=profile.json --cap-drop=ALL
# when running the container

# Set the entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]

# Health check
HEALTHCHECK --interval=5m --timeout=3s \
    CMD pgrep ghactions-updater || exit 1