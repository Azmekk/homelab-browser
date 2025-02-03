# Step 1: Build the Go application
FROM golang:1.23 AS builder

WORKDIR /app

# Copy the go.mod and go.sum files and download dependencies
COPY ./src/main.go ./src/go.mod ./src/go.sum ./
RUN go mod download
RUN go build -o homelab-browser

# Step 2: Create the deployment image
FROM debian:bookworm-slim

WORKDIR /app

# Copy the built application from the builder stage
COPY --from=builder /app/homelab-browser .
COPY ./src/wwwrooot ./app/wwwrooot
COPY ./src/appsettings.json ./app/appsettings.json

# Expose the port the application runs on
EXPOSE 8080

# Command to run the application
CMD ["./homelab-browser"]