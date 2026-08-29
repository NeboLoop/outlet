# Multi-stage Dockerfile for Outlet
# Single all-in-one binary with embedded SvelteKit frontend
# Usage: docker build -t outlet .

# Development stage with Air for hot reloading
FROM golang:1.25-alpine AS development

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git build-base nodejs npm

# Install pnpm globally
RUN npm install -g pnpm@9

# Install Air for hot reloading
RUN go install github.com/air-verse/air@v1.61.5

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build frontend for embedding
RUN cd app && CI=true pnpm install && pnpm run build

# Expose port
EXPOSE 8888

# Use Air for hot reloading in development
CMD ["air"]

# Frontend builder stage
FROM node:20-alpine AS frontend-builder

WORKDIR /app

# Install pnpm
RUN npm install -g pnpm@9

# Copy frontend package files
COPY app/package.json app/pnpm-lock.yaml ./
RUN pnpm install

# Copy frontend source (but exclude node_modules which we just installed)
COPY app/src ./src
COPY app/static ./static
COPY app/svelte.config.js app/vite.config.ts app/tsconfig.json ./

# Generate SvelteKit files and build frontend for production
RUN pnpm exec svelte-kit sync && pnpm run build

# Production builder stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git build-base

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy Go source code and internal packages
COPY *.go ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY app/ ./app/
COPY etc/ ./etc/

# Copy built frontend from frontend-builder
COPY --from=frontend-builder /app/build ./app/build

# Build the all-in-one binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o /app/bin/outlet .

# Final production stage
FROM alpine:latest AS production

RUN apk --no-cache add ca-certificates curl wget tzdata

WORKDIR /app

# Copy the all-in-one binary
COPY --from=builder /app/bin/outlet ./outlet

# Copy configuration files
COPY etc/ ./etc/

# Note: Migrations are embedded in the binary via internal/db/migrations/
# No separate copy needed - they're included in the Go build

# Create necessary directories
RUN mkdir -p /app/certs /app/backups

# Expose ports (80 for HTTP redirect, 443 for HTTPS, 8888 for backend)
EXPOSE 80 443 8888

# Health check on internal backend port
HEALTHCHECK --interval=10s --timeout=3s --retries=3 --start-period=40s \
  CMD curl -sf http://localhost:8888/health || exit 1

# Run the server
CMD ["./outlet", "serve"]
