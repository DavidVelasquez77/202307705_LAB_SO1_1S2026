FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o go-consumer ./apps/go-consumer/cmd/consumer

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/go-consumer /app/go-consumer

CMD ["/app/go-consumer"]