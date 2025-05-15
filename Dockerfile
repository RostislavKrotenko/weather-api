FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o weather-service ./cmd/weather-service

FROM alpine:latest
WORKDIR /root/

COPY --from=builder /app/weather-service .
COPY migrations ./migrations
COPY swagger.yaml ./swagger.yaml
COPY ./public ./public

EXPOSE 8080
ENTRYPOINT ["./weather-service"]