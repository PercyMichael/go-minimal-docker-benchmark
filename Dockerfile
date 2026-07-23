# === STAGE 1: Build the binary ===
FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /app

# Copy module definition
COPY go.mod ./

# Copy source code
COPY . .

# Build statically compiled binary:
# -trimpath: Removes local workspace paths from binary
# -ldflags="-s -w": Strips DWARF & debug symbol table
RUN go build -trimpath -ldflags="-s -w" -o /app/server .

# === STAGE 2: Final ultra-small scratch image ===
FROM scratch

WORKDIR /app

# Copy SSL certificates from builder for HTTPS support
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy static binary from builder stage
COPY --from=builder /app/server /app/server

# Run as non-root user (65534: nobody)
USER 65534:65534

EXPOSE 8080

ENTRYPOINT ["/app/server"]
