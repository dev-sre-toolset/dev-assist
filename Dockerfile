# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

# Download dependencies in a separate layer so they are cached between builds
# unless go.mod / go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source and build.
# VERSION defaults to "dev"; override via --build-arg at publish time.
ARG VERSION=dev
COPY . .
RUN go build \
    -ldflags "-X main.version=${VERSION} -s -w" \
    -o dev-assist \
    .

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM alpine:3.19

# Run as a non-root user.
RUN addgroup -S appgroup && adduser -S -G appgroup appuser

COPY --from=builder /build/dev-assist /usr/local/bin/dev-assist

USER appuser

# The web server listens on this port; keep in sync with deployment/service.yaml.
EXPOSE 8080

# Bind to 0.0.0.0 so container orchestrators can route traffic to the container.
ENTRYPOINT ["dev-assist", "web", "--host", "0.0.0.0", "--port", "8080"]
