FROM golang:alpine AS builder

# Install build dependencies for SQLite (CGO required)
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Enable CGO for SQLite driver
ENV CGO_ENABLED=1
RUN go build -o main ./cmd/server

FROM alpine:latest

# Install SQLite runtime library
RUN apk add --no-cache sqlite

WORKDIR /app
COPY --from=builder /app/main .

EXPOSE 3000

CMD ["./main"]
