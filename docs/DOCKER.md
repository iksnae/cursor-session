# Docker Development Environment

This guide explains how to use Docker for local development of cursor-session, allowing you to work in an isolated Ubuntu environment that matches GitHub Actions, with cursor-agent running inside the container.

## Overview

The Docker setup provides:

- **Isolated environment**: Development happens in a container, not affecting your host system
- **Ubuntu-based**: Matches GitHub Actions environment for consistent testing
- **cursor-agent integration**: cursor-agent runs inside the container, creating storage that our CLI tool can interact with
- **Hot-reload**: Source code is mounted as a volume, so changes are immediately available
- **Ephemeral storage**: cursor-agent storage is ephemeral (like GitHub Actions) - each container run starts fresh, avoiding false positives in testing

## Prerequisites

- Docker and Docker Compose installed
- Git (for cloning the repository)
- Cursor API Key (optional but recommended for cursor-agent authentication)
  - Get your API key from: https://cursor.sh/settings
  - Set it as an environment variable: `export CURSOR_API_KEY=your-api-key`
  - Or create a `.env` file with: `CURSOR_API_KEY=your-api-key`

## Quick Start

1. **Start the development container:**

   ```bash
   make docker-dev
   ```

2. **Access the container shell:**

   ```bash
   make docker-shell
   ```

3. **Inside the container, build and test:**

   ```bash
   # Build the CLI tool
   make build

   # Run tests
   make test

   # Test with cursor-agent
   ./cursor-session healthcheck
   ```

## Docker Services

The `docker-compose.yml` defines three services:

### `dev` - Development Container

- Interactive development environment
- Source code mounted for hot-reload
- Ephemeral cursor-agent storage (created fresh each run, like GitHub Actions)
- Clean environment for each container start

### `test` - Test Runner

- Runs all tests in container environment
- Uses same storage setup as dev
- Clean environment for each run

### `build` - Build Service

- Builds the binary in container
- Output available on host at `./cursor-session`

## Working with cursor-agent

### Setting Up CURSOR_API_KEY

The `CURSOR_API_KEY` environment variable is automatically passed from your host to the container. Set it before starting the container:

**Option 1: Export in your shell**

```bash
export CURSOR_API_KEY=your-api-key-here
make docker-dev
```

**Option 2: Use .env file**

```bash
# Create .env file (copy from .env.example)
echo "CURSOR_API_KEY=your-api-key-here" > .env
make docker-dev
```

**Option 3: Pass directly**

```bash
CURSOR_API_KEY=your-api-key-here make docker-dev
```

The API key will be available in all container services (dev, test, build) and allows cursor-agent to authenticate without interactive login.

### Initial Setup

1. **Set CURSOR_API_KEY (recommended):**

   ```bash
   export CURSOR_API_KEY=your-api-key-here
   ```

2. **Start the development container:**

   ```bash
   make docker-dev
   make docker-shell
   ```

3. **Verify API key is set in container:**

   ```bash
   echo $CURSOR_API_KEY  # Should show your API key
   ```

4. **Verify cursor-agent is available:**

   ```bash
   # cursor-agent is installed directly via the official CLI installation
   cursor-agent --version
   ```

5. **Create a test session (with API key, no login needed):**
   ```bash
   # This will create a session in the container's storage
   # Note: You may see superuser warnings - these are expected in Docker containers
   cursor-agent -p "hello" --model auto --print
   ```

### Testing Storage Behavior

1. **Validate storage setup:**

   ```bash
   # Run the storage validation script
   bash docker/test-storage.sh
   ```

2. **Use the CLI tool to interact with cursor-agent storage:**

   ```bash
   # Build the tool
   make build

   # Check health
   ./cursor-session healthcheck

   # List sessions
   ./cursor-session list

   # Snoop to find storage paths
   ./cursor-session snoop

   # Seed database with cursor-agent (requires CURSOR_API_KEY)
   # This will invoke cursor-agent with a "hello" message
   ./cursor-session snoop --hello
   ```

### Storage Locations

In the container, cursor-agent storage is located at:

- Primary: `/root/.config/cursor/chats/`
- Fallback: `/root/.cursor/chats/`

**Note**: Storage is ephemeral (like GitHub Actions) - directories are created by cursor-agent when it runs, and are not persisted across container restarts. This ensures clean testing without false positives from previous runs.

## Development Workflow

### Daily Development

1. **Start container:**

   ```bash
   make docker-dev
   ```

2. **Access shell:**

   ```bash
   make docker-shell
   ```

3. **Make code changes** (on host - files are mounted)

4. **Test changes** (in container):

   ```bash
   make test
   ./cursor-session healthcheck
   ```

5. **Build and test CLI:**
   ```bash
   make build
   ./cursor-session list
   ```

### Running Tests

```bash
# Run tests in Docker
make docker-test

# Or inside the container
make docker-shell
make test
```

### Building Binaries

```bash
# Build in Docker (output on host)
make docker-build

# Or inside container
make docker-shell
make build
```

## Environment Variables

Create a `.env` file (copy from `.env.example`) to customize:

```bash
# Cursor API Key for authentication
CURSOR_API_KEY=your-api-key-here

# Go version
GO_VERSION=1.23.0

# Storage path
CURSOR_STORAGE_PATH=/root/.config/cursor/chats
```

## Storage Management

### Ephemeral Storage

The container uses ephemeral storage (similar to GitHub Actions):

- Storage directories are **not** pre-created
- cursor-agent creates storage directories when it first runs
- Each container run starts with a clean slate
- No persistent volumes - storage is created fresh each time

This ensures:

- No false positives from previous test runs
- Consistent behavior with CI/CD environments
- Clean testing environment every time

### Inspect Storage

```bash
# Access container
make docker-shell

# Check storage directory
ls -la ~/.config/cursor/chats/

# Count sessions
find ~/.config/cursor/chats -name "store.db" | wc -l
```

## Troubleshooting

### cursor-agent Not Found

If cursor-agent is not available:

1. **Check installation:**

   ```bash
   make docker-shell
   which cursor-agent
   ```

2. **Install manually:**

   ```bash
   curl https://cursor.com/install -fsS | bash
   ```

3. **Check PATH:**
   ```bash
   echo $PATH
   ls -la ~/.local/bin/
   ```

### Authentication Issues

If cursor-agent requires authentication:

1. **Set API key on host (recommended):**

   ```bash
   # Export before starting container
   export CURSOR_API_KEY=your-key
   make docker-dev

   # Or use .env file
   echo "CURSOR_API_KEY=your-key" > .env
   make docker-dev
   ```

2. **Verify API key is in container:**

   ```bash
   make docker-shell
   echo $CURSOR_API_KEY  # Should show your key
   ```

3. **Or use interactive login (if API key not available):**
   ```bash
   make docker-shell
   cursor-agent login
   ```

### How cursor-agent Works in Docker

The `cursor-agent` CLI is installed directly using the official installation script (`curl https://cursor.com/install -fsS | bash`). The Docker setup:

1. **Installs cursor-agent CLI** via the official installation script during image build
2. **Places cursor-agent** at `/root/.local/bin/cursor-agent`
3. **Makes it available** in PATH automatically

**Note:** You may see warnings about running as superuser - these are expected in Docker containers running as root and can be safely ignored.

### Storage Not Detected

If the CLI tool can't find cursor-agent storage:

1. **Verify storage exists:**

   ```bash
   make docker-shell
   ls -la ~/.config/cursor/chats/
   ```

2. **Create a test session:**

   ```bash
   cursor-agent -p "test" --model auto --print
   ```

3. **Check paths:**

   ```bash
   ./cursor-session snoop
   ```

4. **Use --hello flag to seed:**
   ```bash
   ./cursor-session snoop --hello
   ```

### Container Won't Start

1. **Check Docker is running:**

   ```bash
   docker ps
   ```

2. **Rebuild image:**

   ```bash
   docker-compose build --no-cache
   ```

3. **Clean and restart:**
   ```bash
   make docker-clean
   make docker-dev
   ```

### Build Failures

If builds fail in container:

1. **Check Go version:**

   ```bash
   make docker-shell
   go version
   ```

2. **Verify dependencies:**

   ```bash
   go mod download
   go mod verify
   ```

3. **Clean and rebuild:**
   ```bash
   make clean
   make docker-build
   ```

## Advanced Usage

### Custom docker-compose Override

Create `docker-compose.override.yml` (see `docker/docker-compose.override.yml.example`) to customize:

- Additional volume mounts
- Environment variables
- Port mappings
- Service configurations

### Running Specific Tests

```bash
make docker-shell
go test ./internal/... -v
go test -run TestSpecific ./...
```

### Debugging

```bash
# Access container with debugging tools
make docker-shell

# Install additional tools
apt-get update && apt-get install -y vim curl

# Check logs
docker-compose logs dev
```

## CI/CD Integration

The Docker environment matches GitHub Actions (Ubuntu 22.04), making it ideal for:

- Testing storage behavior in Linux environment
- Reproducing CI issues locally
- Validating path detection on Linux
- Testing cursor-agent integration

## Best Practices

1. **Use named volumes** for cursor-agent storage (already configured)
2. **Reset storage** when testing different scenarios: `make docker-reset`
3. **Keep .env file** out of version control (use .env.example)
4. **Test in container** before committing changes
5. **Use healthcheck** to verify setup: `./cursor-session healthcheck`

## Additional Resources

- [Main README](../README.md) - General project information
- [Usage Guide](USAGE.md) - CLI command reference
- [Testing Guide](TESTING.md) - Testing strategy
