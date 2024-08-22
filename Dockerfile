# Use a minimal base image with Go
FROM golang:1.22-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy the Go source code
COPY . .

# Build the Go binary
RUN go mod tidy
RUN go build -o admission-controller .

# Final image
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the compiled binary from the builder
COPY --from=builder /app/admission-controller /app/admission-controller

# Add TLS certificates
COPY tls.crt /etc/webhook/certs/tls.crt
COPY tls.key /etc/webhook/certs/tls.key

# Expose port 8443
EXPOSE 8443

# Run the binary
ENTRYPOINT ["/app/admission-controller"]