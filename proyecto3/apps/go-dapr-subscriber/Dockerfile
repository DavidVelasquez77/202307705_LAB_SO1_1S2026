FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app/apps/go-dapr-subscriber
COPY apps/go-dapr-subscriber/ .
RUN go mod tidy
RUN go build -o /go-dapr-subscriber ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /go-dapr-subscriber /app/go-dapr-subscriber
EXPOSE 8083
CMD ["/app/go-dapr-subscriber"]
