# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY services/search/ ./services/search/

WORKDIR /app/services/search
RUN CGO_ENABLED=0 GOOS=linux go build -o search-service .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

COPY --from=builder /app/services/search/search-service .

EXPOSE 8080

CMD ["./search-service"]
