# ─── Stage 1: Build ───────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

# Install git (needed for go mod download with VCS deps)
RUN apk add --no-cache git

WORKDIR /app

# Cache dependency downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a statically-linked binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/user-service ./cmd

# ─── Stage 2: Runtime ─────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy the binary and migration files
COPY --from=builder /app/user-service .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

ENTRYPOINT ["/app/user-service"]
