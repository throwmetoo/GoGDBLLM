FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN apk add --no-cache gcc musl-dev
RUN go build -o gogdbllm ./cmd/gogdbllm

FROM alpine:latest

RUN apk add --no-cache gdb

WORKDIR /app

COPY --from=builder /app/gogdbllm .
COPY web/ ./web/

# Create uploads directory with proper permissions
RUN mkdir -p /app/uploads && chmod 777 /app/uploads

EXPOSE 8080

CMD ["./gogdbllm"] 