# Backend Dockerfile
# 1. Build stage
FROM golang:1.22-alpine AS build
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with security hardening flags
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags='-s -w -extldflags "-static"' -a -installsuffix cgo -o server .

# 2. Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

# Copy ca-certificates from build stage
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app
COPY --from=build --chmod=755 /app/server /app/server

EXPOSE 8080
USER nonroot
ENTRYPOINT ["/app/server"]
