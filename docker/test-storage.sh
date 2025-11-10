#!/bin/bash
set -e

echo "Testing cursor-agent storage setup..."

# Check if cursor-agent is available
if ! command -v cursor-agent &> /dev/null; then
    echo "❌ cursor-agent not found in PATH"
    exit 1
fi

echo "✓ cursor-agent found: $(which cursor-agent)"
cursor-agent --version || true

# Check storage directories
STORAGE_PATHS=(
    "${HOME}/.config/cursor/chats"
    "${HOME}/.cursor/chats"
)

for path in "${STORAGE_PATHS[@]}"; do
    if [ -d "$path" ]; then
        echo "✓ Storage directory exists: $path"
        # Count store.db files
        DB_COUNT=$(find "$path" -name "store.db" 2>/dev/null | wc -l)
        if [ "$DB_COUNT" -gt 0 ]; then
            echo "  → Found $DB_COUNT store.db file(s)"
        else
            echo "  → No store.db files found (this is OK if cursor-agent hasn't created sessions yet)"
        fi
    else
        echo "⚠ Storage directory does not exist: $path"
    fi
done

# Test CLI tool path detection
if [ -f /workspace/cursor-session ]; then
    echo ""
    echo "Testing CLI tool..."
    /workspace/cursor-session healthcheck || {
        echo "⚠ Healthcheck failed (this may be expected if no sessions exist)"
    }

    echo ""
    echo "Testing path detection..."
    /workspace/cursor-session snoop || {
        echo "⚠ Snoop command failed"
    }
else
    echo ""
    echo "⚠ CLI tool not built yet. Run 'make docker-build' first"
fi

echo ""
echo "✓ Storage validation complete"
