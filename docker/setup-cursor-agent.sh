#!/bin/bash
set -e

echo "Setting up cursor-agent..."

# Check if cursor-agent is installed directly
if command -v cursor-agent &> /dev/null; then
    echo "✓ cursor-agent CLI is installed"
    cursor-agent --version || true
else
    echo "❌ cursor-agent not found in PATH"
    echo "cursor-agent should be installed at ~/.local/bin/cursor-agent"
    echo "Install it using: curl https://cursor.com/install -fsS | bash"
    exit 1
fi

# Note: Storage directories will be created by cursor-agent when it runs
# We don't pre-create them to avoid false positives in testing
echo "✓ cursor-agent ready (storage directories will be created on first use)"
