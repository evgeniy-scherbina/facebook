FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY services/mul/ ./services/mul/
WORKDIR /app/services/mul
RUN CGO_ENABLED=0 GOOS=linux go build -o mul-service main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/mul/mul-service .
EXPOSE 8080
CMD ["./mul-service"]
