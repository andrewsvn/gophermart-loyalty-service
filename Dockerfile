FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o gophermart ./cmd/gophermart

FROM alpine:latest
COPY --from=builder /app/gophermart /usr/local/bin/gophermart

# if HTTPS needed
RUN apk add --no-cache ca-certificates

WORKDIR /app

EXPOSE 8080
ENTRYPOINT ["gophermart"]