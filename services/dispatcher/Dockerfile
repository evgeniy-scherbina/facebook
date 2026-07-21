# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# The dispatcher imports internal/ packages (hashring, discovery, proxy),
# so copy those in addition to the service's own source.
COPY internal/ ./internal/
COPY services/dispatcher/ ./services/dispatcher/

# Build the application
WORKDIR /app/services/dispatcher
RUN CGO_ENABLED=0 GOOS=linux go build -o dispatcher .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

COPY --from=builder /app/services/dispatcher/dispatcher .

EXPOSE 8080

CMD ["./dispatcher"]
