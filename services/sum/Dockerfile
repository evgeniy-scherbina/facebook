FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY services/sum/ ./services/sum/
WORKDIR /app/services/sum
RUN CGO_ENABLED=0 GOOS=linux go build -o sum-service main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/sum/sum-service .
EXPOSE 8080
CMD ["./sum-service"]
