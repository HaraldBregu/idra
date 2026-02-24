#!/usr/bin/env bash
set -euo pipefail

# Idra installer for Linux and macOS
# Usage: curl -fsSL https://example.com/install.sh | bash

REPO="your-org/idra"
INSTALL_DIR="/usr/local/bin"
BINARY="idra"

info()  { printf '\033[1;34m[info]\033[0m  %s\n' "$*"; }
error() { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; exit 1; }

# Detect OS and architecture
detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$OS" in
        linux)  OS="linux" ;;
        darwin) OS="darwin" ;;
        *)      error "Unsupported OS: $OS" ;;
    esac

    case "$ARCH" in
        x86_64|amd64)  ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *)             error "Unsupported architecture: $ARCH" ;;
    esac

    info "Detected platform: ${OS}/${ARCH}"
}

# Get latest release version from GitHub
get_latest_version() {
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        error "Could not determine latest version"
    fi
    info "Latest version: ${VERSION}"
}

# Download and install the binary
download_and_install() {
    local url="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}-${OS}-${ARCH}"
    local tmp
    tmp="$(mktemp)"

    info "Downloading ${url}..."
    curl -fsSL -o "$tmp" "$url"
    chmod +x "$tmp"

    info "Installing to ${INSTALL_DIR}/${BINARY}..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "$tmp" "${INSTALL_DIR}/${BINARY}"
    else
        sudo mv "$tmp" "${INSTALL_DIR}/${BINARY}"
    fi

    info "Installed ${BINARY} ${VERSION} to ${INSTALL_DIR}/${BINARY}"
}

# Install and start the service
setup_service() {
    info "Installing service..."
    "${INSTALL_DIR}/${BINARY}" service install

    info "Starting service..."
    "${INSTALL_DIR}/${BINARY}" service start

    info "Done! Idra is running. Opening browser..."
    sleep 1

    # Try to open browser
    if command -v xdg-open &>/dev/null; then
        xdg-open "http://127.0.0.1:8080" 2>/dev/null || true
    elif command -v open &>/dev/null; then
        open "http://127.0.0.1:8080" 2>/dev/null || true
    fi
}

main() {
    info "Installing Idra..."
    detect_platform
    get_latest_version
    download_and_install
    setup_service
    info "Installation complete!"
}

main "$@"
