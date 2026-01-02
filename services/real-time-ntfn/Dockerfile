# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY services/real-time-ntfn/ ./services/real-time-ntfn/

# Build the application
WORKDIR /app/services/real-time-ntfn
RUN CGO_ENABLED=0 GOOS=linux go build -o real-time-ntfn-service main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/services/real-time-ntfn/real-time-ntfn-service .

EXPOSE 8081

CMD ["./real-time-ntfn-service"]

