# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod ./

# Download dependencies (if any, though we only use standard library)
RUN go mod download

# Copy source code
COPY . .

# Build compiled binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o opencode-proxy main.go

# Production stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/opencode-proxy .

EXPOSE 20128

ENV PORT=20128

CMD ["./opencode-proxy"]
