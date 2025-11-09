#!/bin/bash

# Installation script for cursor-session CLI
# Supports macOS (zsh) and Linux (bash/zsh)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Welcome message
echo ""
echo -e "${CYAN}${BOLD}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}${BOLD}║                                                          ║${NC}"
echo -e "${CYAN}${BOLD}║${NC}  ${GREEN}${BOLD}Welcome to cursor-session CLI!${NC}${CYAN}${BOLD}                      ║${NC}"
echo -e "${CYAN}${BOLD}║                                                          ║${NC}"
echo -e "${CYAN}${BOLD}║${NC}  Extract and export your Cursor IDE chat sessions   ${CYAN}${BOLD}║${NC}"
echo -e "${CYAN}${BOLD}║${NC}  in multiple formats (JSONL, Markdown, YAML, JSON) ${CYAN}${BOLD}║${NC}"
echo -e "${CYAN}${BOLD}║                                                          ║${NC}"
echo -e "${CYAN}${BOLD}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

echo -e "${GREEN}${BOLD}Installing cursor-session CLI...${NC}"
echo ""

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
    echo -e "${GREEN}${BOLD}✓ Installation complete!${NC}"
    echo ""
    echo -e "${CYAN}${BOLD}Quick Start:${NC}"
    echo ""
    echo -e "  ${YELLOW}cursor-session list${NC}              ${BLUE}# List all your chat sessions${NC}"
    echo -e "  ${YELLOW}cursor-session show <id>${NC}          ${BLUE}# View a specific session${NC}"
    echo -e "  ${YELLOW}cursor-session export --format md${NC} ${BLUE}# Export sessions as Markdown${NC}"
    echo ""
    echo -e "${CYAN}${BOLD}For more information:${NC}"
    echo -e "  ${YELLOW}cursor-session --help${NC}            ${BLUE}# See all available commands${NC}"
    echo ""
    echo -e "${GREEN}Installed version:${NC}"
    cursor-session --version 2>/dev/null || true
    echo ""
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
