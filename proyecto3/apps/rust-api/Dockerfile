FROM rust:1.86 AS builder

WORKDIR /app
COPY apps/rust-api ./apps/rust-api
WORKDIR /app/apps/rust-api
RUN cargo build --release

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/apps/rust-api/target/release/rust-api /app/rust-api

EXPOSE 8080
CMD ["/app/rust-api"]