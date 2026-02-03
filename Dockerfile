FROM golang:1.21-alpine
WORKDIR /app
COPY main.go .
RUN go build -o servidor-api main.go
ENTRYPOINT ["./servidor-api"]
