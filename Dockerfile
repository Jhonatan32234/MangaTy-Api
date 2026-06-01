# Cambiamos a la versión 1.25-alpine
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /mangaty-api ./cmd/api

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /mangaty-api .
COPY docs/ ./docs/

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./mangaty-api"]