FROM golang:1.25.0-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/taskservice ./cmd/api

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/taskservice /app/taskservice
# Копируем папку с миграциями
COPY --from=builder /src/migrations /app/migrations

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

CMD ["/app/taskservice"]