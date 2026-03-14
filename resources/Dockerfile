# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN git clone https://github.com/rnts/blockchain-vpn.git .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o bcvpn .

# Runtime stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/bcvpn .
CMD ["./bcvpn", "version"]
