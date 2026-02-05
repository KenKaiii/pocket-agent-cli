#!/bin/bash
set -e

# Pocket CLI Local Installer (for development)
# For production install, use: curl -fsSL https://raw.githubusercontent.com/KenKaiii/pocket-agent-cli/main/scripts/install.sh | bash

INSTALL_DIR="$HOME/.local/bin"
BINARY_NAME="pocket"

echo ""
echo "╔═══════════════════════════════════════╗"
echo "║   Pocket CLI Local Install (Dev)      ║"
echo "╚═══════════════════════════════════════╝"
echo ""

# Create install directory
mkdir -p "$INSTALL_DIR"

# Build the binary
echo "Building..."
go build -ldflags "-s -w" -o "$INSTALL_DIR/$BINARY_NAME" ./cmd/pocket

# Check if install dir is in PATH and add if needed
PATH_EXPORT="export PATH=\"\$PATH:$INSTALL_DIR\""
COMMENT="# Pocket CLI"
ADDED_TO=""

add_to_config() {
    local file="$1"
    local name="$2"
    if [ -f "$file" ]; then
        if ! grep -q "$INSTALL_DIR" "$file" 2>/dev/null; then
            echo "" >> "$file"
            echo "$COMMENT" >> "$file"
            echo "$PATH_EXPORT" >> "$file"
            ADDED_TO="$ADDED_TO $name"
        fi
    fi
}

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "Configuring PATH..."
    add_to_config "$HOME/.zshrc" ".zshrc"
    add_to_config "$HOME/.bashrc" ".bashrc"
    add_to_config "$HOME/.bash_profile" ".bash_profile"
    add_to_config "$HOME/.profile" ".profile"
    if [ -n "$ADDED_TO" ]; then
        echo "  Added to:$ADDED_TO"
    fi
fi

echo ""
echo "════════════════════════════════════════"
echo "✅ Pocket CLI installed to $INSTALL_DIR/$BINARY_NAME"
echo "════════════════════════════════════════"
echo ""
echo "Restarting shell to apply PATH changes..."
echo ""

# Restart shell
CURRENT_SHELL=$(basename "$SHELL")
case "$CURRENT_SHELL" in
    zsh)
        exec zsh -l
        ;;
    bash)
        exec bash -l
        ;;
    *)
        echo "Please restart your terminal, then run:"
        echo "  pocket commands"
        ;;
esac
