# Backend Dockerfile
# 1. Build stage
FROM golang:1.22-alpine AS build

# Install ca-certificates and dependencies
RUN apk --no-cache add ca-certificates git curl

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (SQLC code is pre-generated and committed)
COPY . .

# Build with security hardening flags  
RUN CGO_ENABLED=0 \
    go build -ldflags='-s -w -extldflags "-static"' -a -installsuffix cgo -o server ./cmd && \
    chmod +x server

# 2. Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

# Copy ca-certificates from build stage
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app
COPY --from=build /app/server /app/server

EXPOSE 8080
USER nonroot
ENTRYPOINT ["/app/server"]
