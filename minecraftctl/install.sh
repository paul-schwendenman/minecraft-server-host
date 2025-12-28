#!/usr/bin/env bash
# Installs the latest release of minecraftctl from GitHub
# Repo: https://github.com/paul-schwendenman/minecraft-server-host
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/paul-schwendenman/minecraft-server-host/master/minecraftctl/install.sh | bash
#
# Options:
#   Set INSTALL_DIR to change binary location (default: /usr/local/bin)
#   Set INSTALL_MAN=1 to install man pages
#   Set INSTALL_COMPLETIONS=1 to install shell completions

set -euo pipefail

REPO="paul-schwendenman/minecraft-server-host"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
INSTALL_MAN="${INSTALL_MAN:-0}"
INSTALL_COMPLETIONS="${INSTALL_COMPLETIONS:-0}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux)
        OS=linux
        ;;
    darwin)
        OS=darwin
        ;;
    *)
        echo "Unsupported OS: $OS" >&2
        exit 1
        ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        ARCH=amd64
        ;;
    aarch64|arm64)
        ARCH=arm64
        ;;
    *)
        echo "Unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

echo "Detecting system: ${OS}-${ARCH}"

# Get latest release tag from GitHub API
echo "Fetching latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | cut -d '"' -f4)
if [[ -z "$LATEST" ]]; then
    echo "Could not determine latest release tag." >&2
    exit 1
fi

# Extract version (remove minecraftctl- prefix if present)
VERSION="${LATEST#minecraftctl-}"
echo "Latest version: $VERSION"

# Download binary
BINARY_URL="https://github.com/$REPO/releases/download/$LATEST/minecraftctl-${OS}-${ARCH}"
BINARY_TMP=$(mktemp)

echo "Downloading minecraftctl..."
if ! curl -fL "$BINARY_URL" -o "$BINARY_TMP"; then
    echo "Failed to download binary from $BINARY_URL" >&2
    rm -f "$BINARY_TMP"
    exit 1
fi

# Install binary
echo "Installing to ${INSTALL_DIR}/minecraftctl..."
if [[ -w "$INSTALL_DIR" ]]; then
    install -m 755 "$BINARY_TMP" "${INSTALL_DIR}/minecraftctl"
else
    sudo install -m 755 "$BINARY_TMP" "${INSTALL_DIR}/minecraftctl"
fi
rm -f "$BINARY_TMP"

# Install man pages if requested
if [[ "$INSTALL_MAN" == "1" ]]; then
    MAN_URL="https://github.com/$REPO/releases/download/$LATEST/minecraftctl-man.tar.gz"
    MAN_TMP=$(mktemp -d)

    echo "Downloading man pages..."
    if curl -fL "$MAN_URL" -o "${MAN_TMP}/man.tar.gz" 2>/dev/null; then
        tar -xzf "${MAN_TMP}/man.tar.gz" -C "$MAN_TMP"

        MAN_DIR="/usr/local/share/man/man1"
        echo "Installing man pages to ${MAN_DIR}..."
        if [[ -w "$MAN_DIR" ]] || [[ -w "$(dirname "$MAN_DIR")" ]]; then
            mkdir -p "$MAN_DIR"
            install -m 644 "${MAN_TMP}"/man/man1/*.1 "$MAN_DIR/"
        else
            sudo mkdir -p "$MAN_DIR"
            sudo install -m 644 "${MAN_TMP}"/man/man1/*.1 "$MAN_DIR/"
        fi
        echo "Man pages installed. Run 'man minecraftctl' to view."
    else
        echo "Man pages not available for this release (skipping)"
    fi
    rm -rf "$MAN_TMP"
fi

# Install shell completions if requested
if [[ "$INSTALL_COMPLETIONS" == "1" ]]; then
    echo "Downloading shell completions..."

    # Bash completions
    BASH_COMP_DIR="${BASH_COMPLETION_USER_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion/completions}"
    BASH_URL="https://github.com/$REPO/releases/download/$LATEST/minecraftctl.bash"
    if curl -fL "$BASH_URL" -o /tmp/minecraftctl.bash 2>/dev/null; then
        mkdir -p "$BASH_COMP_DIR"
        install -m 644 /tmp/minecraftctl.bash "${BASH_COMP_DIR}/minecraftctl"
        echo "Bash completion installed to ${BASH_COMP_DIR}/minecraftctl"
        rm -f /tmp/minecraftctl.bash
    fi

    # Zsh completions
    ZSH_COMP_DIR="${ZSH_COMPLETION_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/zsh/site-functions}"
    ZSH_URL="https://github.com/$REPO/releases/download/$LATEST/minecraftctl.zsh"
    if curl -fL "$ZSH_URL" -o /tmp/minecraftctl.zsh 2>/dev/null; then
        mkdir -p "$ZSH_COMP_DIR"
        install -m 644 /tmp/minecraftctl.zsh "${ZSH_COMP_DIR}/_minecraftctl"
        echo "Zsh completion installed to ${ZSH_COMP_DIR}/_minecraftctl"
        rm -f /tmp/minecraftctl.zsh
    fi

    # Fish completions
    FISH_COMP_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/fish/completions"
    FISH_URL="https://github.com/$REPO/releases/download/$LATEST/minecraftctl.fish"
    if curl -fL "$FISH_URL" -o /tmp/minecraftctl.fish 2>/dev/null; then
        mkdir -p "$FISH_COMP_DIR"
        install -m 644 /tmp/minecraftctl.fish "${FISH_COMP_DIR}/minecraftctl.fish"
        echo "Fish completion installed to ${FISH_COMP_DIR}/minecraftctl.fish"
        rm -f /tmp/minecraftctl.fish
    fi
fi

echo ""
echo "minecraftctl installed successfully!"
echo "  Version: $VERSION"
echo "  Binary:  ${INSTALL_DIR}/minecraftctl"
echo ""
echo "Run 'minecraftctl --help' to get started."

if [[ "$INSTALL_MAN" != "1" ]]; then
    echo ""
    echo "Tip: Run with INSTALL_MAN=1 to also install man pages"
fi

if [[ "$INSTALL_COMPLETIONS" != "1" ]]; then
    echo "Tip: Run with INSTALL_COMPLETIONS=1 to install shell completions"
fi
