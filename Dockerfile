FROM golang:1.24.5 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o internly .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    libc6 \
    && rm -rf /var/lib/apt/lists/*

RUN apt-get update && apt-get install -y ca-certificates

WORKDIR /app

COPY --from=builder /app/internly .

RUN chmod +x ./internly

CMD ["./internly"]
