FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/api/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates && adduser -D -u 10001 app

WORKDIR /app
COPY --from=builder /app/main .

USER app
EXPOSE 8083

CMD ["./main"]
