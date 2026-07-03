FROM golang:1.25-alpine

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git gcc musl-dev

# Install air for hot reload
RUN go install github.com/air-verse/air@v1.52.3

# Create non-root user for security
# Using UID 1000 which is commonly available and avoids conflicts
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Update modules and vendor dependencies so they are available even without network
RUN go mod tidy && go mod vendor

# Create tmp directory for air, clear any old binaries, and set ownership
# Important: rm -rf ensures any cached files from previous builds are removed
RUN rm -rf /app/tmp && \
    mkdir -p /app/tmp && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run with air for hot reload in development using vendor
CMD ["air", "-c", ".air.toml"]
