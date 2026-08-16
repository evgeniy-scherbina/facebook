# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# post imports internal/tokenize, so the internal packages must be copied too.
COPY internal/ ./internal/
COPY services/post/ ./services/post/

WORKDIR /app/services/post
RUN CGO_ENABLED=0 GOOS=linux go build -o post-service .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

COPY --from=builder /app/services/post/post-service .

EXPOSE 8080

CMD ["./post-service"]
