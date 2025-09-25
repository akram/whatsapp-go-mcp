# Use the official Go image as the base image
FROM golang:1.21-alpine AS builder

# Install necessary packages
RUN apk add --no-cache git sqlite-dev gcc musl-dev

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

# Use a minimal image for the final stage
FROM alpine:latest

# Install necessary packages
RUN apk --no-cache add ca-certificates sqlite

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
