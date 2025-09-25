# Use UBI9 with Go as the base image for building
FROM registry.redhat.io/ubi9/go-toolset:latest AS builder

# Switch to root to install packages
USER root

# Install necessary packages
RUN dnf install -y git sqlite-devel gcc && dnf clean all

# Switch back to the default user (1001) for security
USER 1001

# Set the working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o whatsapp-mcp-server .

# Use UBI9 minimal for the final stage
FROM registry.redhat.io/ubi9/ubi-minimal:latest

# Install necessary packages
RUN microdnf install -y sqlite ca-certificates && microdnf clean all

# Create app directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/whatsapp-mcp-server .

# Create directories for data and media
RUN mkdir -p /app/data /app/media /app/qr_codes

# Set environment variables
ENV PORT=8080
ENV WHATSAPP_DB_PATH=/app/data/whatsapp.db
ENV WHATSAPP_MEDIA_DIR=/app/media
ENV QR_CODE_DIR=/app/qr_codes
ENV LOG_LEVEL=info

# Expose the port
EXPOSE 8080

# Run the application
CMD ["./whatsapp-mcp-server"]
