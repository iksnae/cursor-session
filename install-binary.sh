#!/bin/bash

# Simple binary installer for cursor-session CLI
# Downloads and installs pre-built binary for your platform
# Usage: curl -fsSL https://raw.githubusercontent.com/iksnae/cursor-session/main/install-binary.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Configuration
REPO="iksnae/cursor-session"
BINARY_NAME="cursor-session"
INSTALL_DIR="$HOME/.local/bin"
LATEST_VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || echo "v0.0.2")

# Detect OS and architecture
OS=$(uname -s)
ARCH=$(uname -m)

# Map OS
case "$OS" in
    Linux)  OS_NAME="linux" ;;
    Darwin) OS_NAME="darwin" ;;
    *)      echo -e "${RED}Error: Unsupported OS: $OS${NC}" >&2; exit 1 ;;
esac

# Map architecture
case "$ARCH" in
    x86_64)     ARCH_NAME="amd64" ;;
    amd64)      ARCH_NAME="amd64" ;;
    arm64)      ARCH_NAME="arm64" ;;
    aarch64)    ARCH_NAME="arm64" ;;
    *)          echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}" >&2; exit 1 ;;
esac

# Welcome message
echo ""
echo -e "${CYAN}${BOLD}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}${BOLD}║                                                          ║${NC}"
echo -e "${CYAN}${BOLD}║${NC}  ${GREEN}${BOLD}Installing cursor-session CLI ${NC}${CYAN}${BOLD}                          ║${NC}"
echo -e "${CYAN}${BOLD}║                                                          ║${NC}"
echo -e "${CYAN}${BOLD}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

# Construct download URL
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_VERSION}/${BINARY_NAME}-${LATEST_VERSION#v}-${OS_NAME}-${ARCH_NAME}.tar.gz"
TEMP_DIR=$(mktemp -d)
TEMP_FILE="${TEMP_DIR}/${BINARY_NAME}.tar.gz"

# Create install directory
mkdir -p "$INSTALL_DIR"

# Download binary
echo -e "${YELLOW}Downloading ${BINARY_NAME} ${LATEST_VERSION} for ${OS_NAME}/${ARCH_NAME}...${NC}"
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_FILE"; then
    echo -e "${RED}Error: Failed to download binary${NC}" >&2
    echo -e "${YELLOW}Please check:${NC}"
    echo -e "  - Your internet connection"
    echo -e "  - GitHub releases: https://github.com/${REPO}/releases"
    rm -rf "$TEMP_DIR"
    exit 1
fi

# Extract binary
echo -e "${YELLOW}Extracting...${NC}"
cd "$TEMP_DIR"
tar -xzf "$TEMP_FILE" || {
    echo -e "${RED}Error: Failed to extract archive${NC}" >&2
    rm -rf "$TEMP_DIR"
    exit 1
}

# Verify binary exists
if [ ! -f "$BINARY_NAME" ]; then
    echo -e "${RED}Error: Binary not found in archive${NC}" >&2
    rm -rf "$TEMP_DIR"
    exit 1
fi

# Install binary
echo -e "${YELLOW}Installing to ${INSTALL_DIR}...${NC}"
mv "$BINARY_NAME" "$INSTALL_DIR/${BINARY_NAME}"
chmod +x "$INSTALL_DIR/${BINARY_NAME}"

# Cleanup
rm -rf "$TEMP_DIR"

# Add to PATH if needed
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
        echo -e "${YELLOW}Adding ${INSTALL_DIR} to PATH in ${SHELL_CONFIG}...${NC}"
        echo "" >> "$SHELL_CONFIG"
        echo "# cursor-session CLI" >> "$SHELL_CONFIG"
        echo "$PATH_LINE" >> "$SHELL_CONFIG"
        echo -e "${GREEN}PATH updated${NC}"

        # Source the config file for current session
        if [ "$SHELL_NAME" = "zsh" ]; then
            source "$SHELL_CONFIG" 2>/dev/null || true
        elif [ "$SHELL_NAME" = "bash" ]; then
            source "$SHELL_CONFIG" 2>/dev/null || true
        fi
    else
        echo -e "${GREEN}${INSTALL_DIR} already in PATH${NC}"
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
    echo -e "${GREEN}Installed version:${NC}"
    cursor-session --version 2>/dev/null || true
    echo ""
else
    echo -e "${GREEN}${BOLD}✓ Binary installed to ${INSTALL_DIR}${NC}"
    echo ""
    echo -e "${YELLOW}To use cursor-session, run:${NC}"
    echo ""
    if [ -n "$SHELL_CONFIG" ]; then
        echo "  source $SHELL_CONFIG"
    else
        echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    fi
    echo ""
    echo "Or restart your terminal."
    echo ""
    echo "You can also run directly:"
    echo "  $INSTALL_DIR/cursor-session --help"
    echo ""
fi
