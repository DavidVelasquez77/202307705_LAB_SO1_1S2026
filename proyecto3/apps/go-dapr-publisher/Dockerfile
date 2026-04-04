FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app/apps/go-dapr-publisher
COPY apps/go-dapr-publisher/ .
RUN go get github.com/dapr/go-sdk@v1.9.0
RUN go mod tidy
RUN go build -o /go-dapr-publisher ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /go-dapr-publisher /app/go-dapr-publisher
EXPOSE 8082
CMD ["/app/go-dapr-publisher"]
