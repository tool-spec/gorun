# GoRun

A Go HTTP server that provides a simple interface to run tool-spec containers.

## Prerequisites

- Docker installed and running
- Access to Docker daemon (via socket or TCP)

## Quick Start

1. Download the appropriate binary for your platform
2. Make it executable (Linux/macOS):
   ```bash
   chmod +x gorun
   ```
3. For macOS users: Remove the quarantine attribute (required until we add proper code signing):
   ```bash
   xattr -d com.apple.quarantine gorun-darwin-arm64  # For Apple Silicon
   xattr -d com.apple.quarantine gorun-darwin-amd64  # For Intel Macs
   ```
4. Run the server:
   ```bash
   ./gorun serve
   ```

## Docker Deployment

For production use, we recommend running gorun in a Docker container:

```bash
docker run -d \
  -p 8080:8080 \
  -v /path/to/local/data:/data/gorun \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e GORUN_PORT=8080 \
  -e GORUN_SECRET=your-secret-here \
  -e GORUN_MOUNT_PATH=/data/gorun/mounts \
  -e GORUN_DB=/data/gorun/gorun.db \
  --name gorun \
  tool-spec/gorun:latest
```

## Environment Variables

- `GORUN_PORT` (Optional, default: 8080)
  - Port for the web interface
- `GORUN_SECRET` (Required)
  - Secret key for authentication
- `GORUN_MOUNT_PATH` (Optional)
  - Directory for container mounts
- `GORUN_DB` (Optional)
  - Path to the SQLite database
- `GORUN_PATH` (Optional)
  - Base directory for all gorun data 