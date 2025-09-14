FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o gophermart ./cmd/gophermart

FROM alpine:latest

# dependencies
RUN apk add --no-cache postgresql-client bash

COPY --from=builder /app/gophermart /usr/local/bin/gophermart
COPY /deployment/scripts /app
RUN chmod +x /app/wait-for-postgres.sh

# if HTTPS needed
RUN apk add --no-cache ca-certificates

WORKDIR /app

EXPOSE 8080
ENTRYPOINT ["./wait-for-postgres.sh"]
CMD ["gophermart"]