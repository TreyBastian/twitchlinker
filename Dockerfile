FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o twitchlinker

# Create a minimal production image
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/twitchlinker .

# Create an unprivileged user to run the application
RUN adduser -D appuser
USER appuser

# Expose webhook port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/app/twitchlinker"]