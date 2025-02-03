# Step 1: Build the Go application
FROM golang:1.23 AS builder

WORKDIR /app

# Copy the go.mod and go.sum files and download dependencies
COPY ./src/main.go ./src/go.mod ./src/go.sum ./
COPY ./src/wwwroot ./wwwroot
# Install Dart Sass
RUN apt-get update && apt-get install -y curl
RUN curl -L https://github.com/sass/dart-sass/releases/download/1.83.4/dart-sass-1.83.4-linux-x64.tar.gz -o dart-sass.tar.gz
RUN tar -xzf dart-sass.tar.gz

RUN chmod +x ./dart-sass/sass
RUN chmod +x ./dart-sass/src/dart

# Copy the SCSS files and compile them to CSS
COPY ./src/wwwroot/styles.scss ./src/wwwroot/styles.scss
RUN ./dart-sass/sass ./wwwroot/styles.scss ./wwwroot/styles.css --no-source-map --style=compressed


RUN go mod download
RUN go build -o homelab-browser

# Step 2: Create the deployment image
FROM debian:bookworm-slim

WORKDIR /app

# Copy the built application from the builder stage
COPY --from=builder /app/homelab-browser .
COPY --from=builder /app/wwwroot /app/wwwroot
COPY ./src/settings/appsettings.json /app/appsettings/appsettings.json

RUN chmod +x ./homelab-browser

# Expose the port the application runs on
EXPOSE 8080

# Command to run the application
CMD ["./homelab-browser"]