## Backend Dockerfile with optional Delve debug mode
## Usage:
##   docker build -t backend:prod ./backend                      (production build)
##   docker build --build-arg DEBUG=true -t backend:debug ./backend  (debug build)
## Runtime:
##   Set ENV DEBUG=true to launch under Delve (only works if built with DEBUG=true so dlv is present)

ARG DEBUG=false

# 1. Build stage
FROM golang:1.23-alpine AS build
ARG DEBUG

RUN apk --no-cache add ca-certificates git curl bash
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Install Delve (always present; DEBUG flag only controls whether it's used at runtime)
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Build binary (disable optimizations only if debug)
RUN if [ "$DEBUG" = "true" ]; then \
            echo "Building in DEBUG mode (no optimizations, no inlining)" && \
            CGO_ENABLED=0 go build -gcflags "all=-N -l" -o server ./cmd ; \
        else \
            echo "Building in RELEASE mode" && \
            CGO_ENABLED=0 go build -o server ./cmd ; \
        fi && chmod +x server

# 2. Runtime stage (lightweight Alpine; keeps bash for entrypoint + optional Delve)
FROM alpine:3.20 AS runtime
ARG DEBUG
ENV DEBUG=$DEBUG
RUN apk --no-cache add ca-certificates bash && update-ca-certificates && adduser -D -g '' appuser
WORKDIR /app

COPY --from=build /app/server /app/server
COPY --from=build /go/bin/dlv /usr/local/bin/dlv
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh && chmod +x /usr/local/bin/dlv || true

EXPOSE 8080
# Expose debug port only for documentation; not mapped unless debug compose file used
EXPOSE 2345
USER appuser
ENTRYPOINT ["/entrypoint.sh"]