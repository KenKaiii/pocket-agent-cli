#!/bin/bash
set -e

# Pocket CLI Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/unstablemind/pocket-agent-CLI/main/scripts/install.sh | bash

REPO="KenKaiii/pocket-agent-cli"
BINARY_NAME="pocket"
INSTALL_DIR="$HOME/.local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        linux*) OS="linux" ;;
        darwin*) OS="darwin" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *) error "Unsupported OS: $OS" ;;
    esac

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        armv7*|armv6*) ARCH="arm" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac

    PLATFORM="${OS}_${ARCH}"
    info "Detected platform: $PLATFORM"
}

# Get latest release tag from GitHub
get_latest_version() {
    info "Fetching latest version..."
    VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        error "Could not determine latest version. Check your internet connection."
    fi
    info "Latest version: $VERSION"
}

# Download and install binary
download_and_install() {
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/${BINARY_NAME}_${PLATFORM}.tar.gz"

    info "Downloading from: $DOWNLOAD_URL"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    # Download
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/pocket.tar.gz"; then
        error "Failed to download. Check if release exists for your platform."
    fi

    # Extract
    tar -xzf "$TMP_DIR/pocket.tar.gz" -C "$TMP_DIR"

    # Install
    mkdir -p "$INSTALL_DIR"
    mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

    info "Installed to: $INSTALL_DIR/$BINARY_NAME"
}

# Add to PATH in shell configs
configure_path() {
    if [[ ":$PATH:" == *":$INSTALL_DIR:"* ]]; then
        info "PATH already configured"
        return
    fi

    info "Configuring PATH..."

    PATH_EXPORT="export PATH=\"\$PATH:$INSTALL_DIR\""
    COMMENT="# Pocket CLI"

    # Function to add to config file
    add_to_config() {
        local file="$1"
        if [ -f "$file" ]; then
            if ! grep -q "$INSTALL_DIR" "$file" 2>/dev/null; then
                echo "" >> "$file"
                echo "$COMMENT" >> "$file"
                echo "$PATH_EXPORT" >> "$file"
                info "  Added to $file"
            fi
        fi
    }

    add_to_config "$HOME/.zshrc"
    add_to_config "$HOME/.bashrc"
    add_to_config "$HOME/.bash_profile"
    add_to_config "$HOME/.profile"
}

# Main installation
main() {
    echo ""
    echo "╔═══════════════════════════════════════╗"
    echo "║       Pocket CLI Installer            ║"
    echo "╚═══════════════════════════════════════╝"
    echo ""

    detect_platform
    get_latest_version
    download_and_install
    configure_path

    echo ""
    echo "════════════════════════════════════════"
    echo -e "${GREEN}✅ Pocket CLI installed successfully!${NC}"
    echo "════════════════════════════════════════"
    echo ""
    echo "Restarting shell to apply PATH changes..."
    echo ""

    # Restart shell to apply PATH changes
    CURRENT_SHELL=$(basename "$SHELL")
    case "$CURRENT_SHELL" in
        zsh)
            exec zsh -l
            ;;
        bash)
            exec bash -l
            ;;
        *)
            echo "Please restart your terminal or run:"
            echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
            echo ""
            echo "Then try:"
            echo "  pocket commands"
            ;;
    esac
}

main
