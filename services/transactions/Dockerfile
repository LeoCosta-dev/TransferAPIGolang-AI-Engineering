FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./

COPY services/account/go.mod services/account/go.sum services/account/
COPY services/transactions/go.mod services/transactions/go.sum services/transactions/

WORKDIR /app/services/transactions
RUN go mod download

WORKDIR /app
COPY services/transactions services/transactions

WORKDIR /app/services/transactions
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /transactions ./cmd/transactions

FROM alpine:3.22

RUN adduser -D -H -s /sbin/nologin appuser

COPY --from=builder /transactions /transactions

USER appuser

EXPOSE 8089

ENTRYPOINT ["/transactions"]
