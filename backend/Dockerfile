FROM golang:1.25-alpine

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files and vendored dependencies
COPY go.mod go.sum ./
COPY vendor/ vendor/

# Copy the source code
COPY . .

# Build the application using vendored dependencies
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o main ./cmd/api

# Use a smaller image for running
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=0 /app/main .

# Make the binary executable
RUN chmod +x ./main

# Expose the application port
EXPOSE 8080

# Run the binary
CMD ["./main"]
