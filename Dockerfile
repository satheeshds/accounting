FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go install github.com/swaggo/swag/cmd/swag@latest
RUN swag init
RUN CGO_ENABLED=0 GOOS=linux go build -o /accounting -ldflags="-s -w" .

# --- Runtime ---
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY --from=builder /accounting /usr/local/bin/accounting

RUN mkdir -p /data
VOLUME /data

ENV DB_PATH=/data/accounting.db
ENV PORT=80

EXPOSE 80

CMD ["accounting"]
