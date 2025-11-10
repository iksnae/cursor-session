#!/bin/bash
set -e

# Ensure .local/bin exists for cursor-agent (installed during build)
mkdir -p "${HOME}/.local/bin"

# Set up environment variables for cursor-agent
# Note: Storage directories will be created by cursor-agent when it runs
# We don't pre-create them to avoid false positives in testing
export CURSOR_STORAGE_PATH="${HOME}/.config/cursor/chats"
export PATH="${HOME}/.local/bin:${PATH}"

# Verify cursor-agent is directly available (installed via official CLI installation)
if ! command -v cursor-agent &> /dev/null; then
    echo "Warning: cursor-agent not found in PATH. It should be installed at ~/.local/bin/cursor-agent"
fi

# Run the setup script if it exists (always run to ensure cursor-agent is set up)
if [ -f /workspace/docker/setup-cursor-agent.sh ]; then
    echo "Running cursor-agent setup..."
    bash /workspace/docker/setup-cursor-agent.sh || {
        echo "Warning: cursor-agent setup had issues, but continuing..."
    }
fi

# Execute the command passed to the container
exec "$@"
