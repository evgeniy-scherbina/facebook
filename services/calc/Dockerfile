FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY services/calc/ ./services/calc/
WORKDIR /app/services/calc
RUN CGO_ENABLED=0 GOOS=linux go build -o calc-service main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/calc/calc-service .
EXPOSE 8080
CMD ["./calc-service"]
