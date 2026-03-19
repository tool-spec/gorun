FROM golang:1.24-alpine AS builder

# Install Node.js, npm, make, and git for the build process
RUN apk add --no-cache nodejs npm make git

WORKDIR /app
COPY . .

# Run the build process using Makefile
RUN make build

FROM alpine

RUN mkdir -p /app
COPY --from=builder /app/gorun /app/gorun
RUN chmod +x /app/gorun

# Create directory for persistent data
RUN mkdir -p /data/gorun

WORKDIR /app

# Set environment variables for data storage
ENV GORUN_PATH=/data/gorun
ENV GORUN_DB=/data/gorun/gorun.db

# Set environment variable, to accept all hosts within a container
ENV GORUN_HOST=0.0.0.0

# Expose the volume for persistent data
VOLUME /data/gorun

CMD ["./gorun", "serve"]
