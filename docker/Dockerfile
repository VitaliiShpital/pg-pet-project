# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE_NAME
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/service ./cmd/${SERVICE_NAME}

# Run stage
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/service /app/service

ENTRYPOINT ["/app/service"]
