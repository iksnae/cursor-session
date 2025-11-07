#!/bin/bash

# Installation script for cursor-session CLI
# Supports macOS (zsh) and Linux (bash/zsh)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Installing cursor-session CLI...${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed. Please install Go first.${NC}"
    echo "Visit: https://golang.org/doc/install"
    exit 1
fi

# Detect OS
OS="$(uname -s)"
ARCH="$(uname -m)"

# Determine install directory (always user-local for simplicity)
INSTALL_DIR="$HOME/.local/bin"

# Create install directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Build the binary
echo -e "${YELLOW}Building cursor-session...${NC}"
cd "$(dirname "$0")"
go build -buildvcs=false -o cursor-session .

# Install the binary
echo -e "${YELLOW}Installing to $INSTALL_DIR...${NC}"
cp cursor-session "$INSTALL_DIR/cursor-session"
chmod +x "$INSTALL_DIR/cursor-session"

# Detect shell and config file
SHELL_NAME="$(basename "$SHELL")"
SHELL_CONFIG=""

if [ "$SHELL_NAME" = "zsh" ]; then
    SHELL_CONFIG="$HOME/.zshrc"
elif [ "$SHELL_NAME" = "bash" ]; then
    SHELL_CONFIG="$HOME/.bashrc"
else
    # Try to detect from common shells
    if [ -f "$HOME/.zshrc" ]; then
        SHELL_CONFIG="$HOME/.zshrc"
        SHELL_NAME="zsh"
    elif [ -f "$HOME/.bashrc" ]; then
        SHELL_CONFIG="$HOME/.bashrc"
        SHELL_NAME="bash"
    fi
fi

# Add to PATH if not already present
if [ -n "$SHELL_CONFIG" ]; then
    PATH_LINE="export PATH=\"\$PATH:$INSTALL_DIR\""
    
    if ! grep -q "$INSTALL_DIR" "$SHELL_CONFIG" 2>/dev/null; then
        echo -e "${YELLOW}Adding $INSTALL_DIR to PATH in $SHELL_CONFIG...${NC}"
        echo "" >> "$SHELL_CONFIG"
        echo "# cursor-session CLI" >> "$SHELL_CONFIG"
        echo "$PATH_LINE" >> "$SHELL_CONFIG"
        echo -e "${GREEN}PATH updated in $SHELL_CONFIG${NC}"
        
        # Source the config file for current session
        if [ "$SHELL_NAME" = "zsh" ]; then
            source "$SHELL_CONFIG" 2>/dev/null || true
        elif [ "$SHELL_NAME" = "bash" ]; then
            source "$SHELL_CONFIG" 2>/dev/null || true
        fi
    else
        echo -e "${GREEN}$INSTALL_DIR already in PATH${NC}"
    fi
else
    echo -e "${YELLOW}Could not detect shell config file.${NC}"
    echo -e "${YELLOW}Please manually add to your shell config:${NC}"
    echo ""
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
fi

# Verify installation
echo ""
if command -v cursor-session &> /dev/null; then
    echo -e "${GREEN}✓ Installation complete!${NC}"
    echo ""
    echo "You can now use 'cursor-session' from anywhere:"
    echo "  cursor-session list"
    echo "  cursor-session show <session-id>"
    echo "  cursor-session export --format jsonl"
    echo ""
    cursor-session --version 2>/dev/null || true
else
    echo -e "${YELLOW}Installation complete, but cursor-session is not in current PATH.${NC}"
    echo -e "${YELLOW}Please run one of the following:${NC}"
    echo ""
    if [ "$SHELL_NAME" = "zsh" ]; then
        echo "  source ~/.zshrc"
    elif [ "$SHELL_NAME" = "bash" ]; then
        echo "  source ~/.bashrc"
    else
        echo "  source ~/.zshrc  # or ~/.bashrc"
    fi
    echo ""
    echo "Or restart your terminal."
    echo ""
    echo "You can also run directly:"
    echo "  $INSTALL_DIR/cursor-session --help"
fi
