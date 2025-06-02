FROM golang:1.23-alpine AS builder

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

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Expose port
EXPOSE 8081

CMD ["./main"]