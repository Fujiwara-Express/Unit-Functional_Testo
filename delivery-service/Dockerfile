# --- Build Stage ---
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o delivery-service main.go

# --- Final Stage ---
FROM alpine:3.21

# CA certificates for TLS connections; tzdata for correct time zones
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Run as non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

COPY --from=builder /app/delivery-service .

EXPOSE 8081

CMD ["./delivery-service"]
