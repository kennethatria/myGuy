FROM golang:1.25-alpine

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

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
