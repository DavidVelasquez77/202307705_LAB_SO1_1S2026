FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o go-ingest ./apps/go-ingest/cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/go-ingest /app/go-ingest

EXPOSE 8081
CMD ["/app/go-ingest"]