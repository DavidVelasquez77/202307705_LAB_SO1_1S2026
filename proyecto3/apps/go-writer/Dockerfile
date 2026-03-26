FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o go-writer ./apps/go-writer/cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/go-writer /app/go-writer

EXPOSE 50051
CMD ["/app/go-writer"]