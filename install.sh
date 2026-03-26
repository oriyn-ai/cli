#!/usr/bin/env bash
set -euo pipefail

REPO="try-bridge/cli"
INSTALL_DIR="/usr/local/bin"

detect_target() {
    local os arch
    os="$(uname -s)"
    arch="$(uname -m)"

    case "$os" in
        Linux)  os="unknown-linux-gnu" ;;
        Darwin) os="apple-darwin" ;;
        *)      echo "Unsupported OS: $os" >&2; exit 1 ;;
    esac

    case "$arch" in
        x86_64|amd64)  arch="x86_64" ;;
        aarch64|arm64) arch="aarch64" ;;
        *)             echo "Unsupported architecture: $arch" >&2; exit 1 ;;
    esac

    echo "${arch}-${os}"
}

main() {
    local target url tmp

    target="$(detect_target)"
    url="https://github.com/${REPO}/releases/latest/download/bridge-${target}"
    tmp="$(mktemp)"

    echo "Detected platform: ${target}"
    echo "Downloading bridge from ${url}..."

    curl -fSL --progress-bar -o "$tmp" "$url"
    chmod +x "$tmp"

    echo "Installing to ${INSTALL_DIR}/bridge (may require sudo)..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "$tmp" "${INSTALL_DIR}/bridge"
    else
        sudo mv "$tmp" "${INSTALL_DIR}/bridge"
    fi

    echo "bridge installed successfully! Run 'bridge --help' to get started."
}

main
