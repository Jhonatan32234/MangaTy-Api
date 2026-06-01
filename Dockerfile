# Cambiamos a la versión 1.25-alpine
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o mangaty-api ./cmd/api/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/mangaty-api .
COPY --from=builder /app/.env .
COPY --from=builder /app/docs ./docs


EXPOSE 8080
CMD ["./mangaty-api"]