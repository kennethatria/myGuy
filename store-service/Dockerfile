FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy all source code first
COPY . .

# Download dependencies and generate go.sum
RUN go mod tidy && go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/api/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/main .

# Create uploads directory with proper permissions
RUN mkdir -p /app/uploads/store && chmod -R 755 /app/uploads

# Expose port
EXPOSE 8081

CMD ["./main"]