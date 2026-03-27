FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o go-stats-api ./apps/go-stats-api/cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/go-stats-api /app/go-stats-api

EXPOSE 8082
CMD ["/app/go-stats-api"]