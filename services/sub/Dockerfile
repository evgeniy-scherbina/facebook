FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY services/sub/ ./services/sub/
WORKDIR /app/services/sub
RUN CGO_ENABLED=0 GOOS=linux go build -o sub-service main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/sub/sub-service .
EXPOSE 8080
CMD ["./sub-service"]
