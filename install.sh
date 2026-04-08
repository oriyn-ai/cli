#!/usr/bin/env bash
set -euo pipefail

REPO="oriyn-ai/cli"
INSTALL_DIR="/usr/local/bin"

detect_target() {
    local os arch
    os="$(uname -s)"
    arch="$(uname -m)"

    case "$os" in
        Linux)  os="linux" ;;
        Darwin) os="darwin" ;;
        *)
            echo "Unsupported OS. Download manually: https://github.com/oriyn-ai/cli/releases/latest" >&2
            exit 1
            ;;
    esac

    case "$arch" in
        x86_64|amd64)  arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)             echo "Unsupported architecture: $arch" >&2; exit 1 ;;
    esac

    echo "${os}-${arch}"
}

checksum_cmd() {
    case "$(uname -s)" in
        Linux)  echo "sha256sum" ;;
        Darwin) echo "shasum -a 256" ;;
    esac
}

resolve_version() {
    if [ -n "${ORIYN_VERSION:-}" ]; then
        echo "$ORIYN_VERSION"
        return
    fi
    # Follow the GitHub "latest" redirect to discover the actual tag
    local location
    location="$(curl -sI "https://github.com/${REPO}/releases/latest" | grep -i ^location: | tr -d '\r')"
    echo "$location" | sed 's|.*/v||'
}

main() {
    local target version binary_name base_url binary_url checksums_url tmp tmp_checksums

    target="$(detect_target)"
    version="$(resolve_version)"
    binary_name="oriyn-${target}"
    base_url="https://github.com/${REPO}/releases/download/v${version}"
    binary_url="${base_url}/${binary_name}"
    checksums_url="${base_url}/checksums.txt"

    echo "Detected platform: ${target}"

    # Check for existing installation
    if command -v oriyn &>/dev/null; then
        local current_version
        current_version="$(oriyn --version | awk '{print $NF}')"
        echo "Found existing oriyn v${current_version} — upgrading to v${version}"
    fi

    echo "Downloading oriyn v${version} from ${binary_url}..."

    tmp="$(mktemp)"
    tmp_checksums="$(mktemp)"
    trap 'rm -f "$tmp" "$tmp_checksums"' EXIT

    curl -fSL --progress-bar -o "$tmp" "$binary_url"
    curl -fsSL -o "$tmp_checksums" "$checksums_url"

    # Verify checksum
    echo "Verifying checksum..."
    local expected actual
    expected="$(grep "${binary_name}" "$tmp_checksums" | awk '{print $1}')"
    actual="$($(checksum_cmd) "$tmp" | awk '{print $1}')"

    if [ "$expected" != "$actual" ]; then
        echo "Checksum verification failed!" >&2
        echo "  Expected: ${expected}" >&2
        echo "  Got:      ${actual}" >&2
        exit 1
    fi
    echo "Checksum verified."

    chmod +x "$tmp"

    echo "Installing to ${INSTALL_DIR}/oriyn (may require sudo)..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "$tmp" "${INSTALL_DIR}/oriyn"
    else
        sudo mv "$tmp" "${INSTALL_DIR}/oriyn"
    fi

    echo "oriyn v${version} installed successfully! Run 'oriyn --help' to get started."
}

main
