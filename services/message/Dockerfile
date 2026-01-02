# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY services/message/ ./services/message/
COPY index.html ./

# Build the application
WORKDIR /app/services/message
RUN CGO_ENABLED=0 GOOS=linux go build -o message-service main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copy the binary and index.html from builder
COPY --from=builder /app/services/message/message-service .
COPY --from=builder /app/index.html .

EXPOSE 8080

CMD ["./message-service"]

