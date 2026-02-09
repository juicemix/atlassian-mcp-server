# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for fetching dependencies
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o atlassian-mcp-server .

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the binary from builder
COPY --from=builder /app/atlassian-mcp-server .

# Copy config example (user should mount their own config.yaml)
COPY config.example.yaml ./config.example.yaml

# Expose port if needed (adjust based on your server configuration)
# EXPOSE 8080

# Run the server
ENTRYPOINT ["./atlassian-mcp-server"]
