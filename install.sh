#!/usr/bin/env bash
set -euo pipefail

REPO="oriyn-ai/cli"
BIN_NAME="oriyn"
SHARE_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/${BIN_NAME}"
BIN_DIR="${XDG_BIN_HOME:-$HOME/.local/bin}"

if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
    BOLD=$'\033[1m'
    DIM=$'\033[2m'
    GREEN=$'\033[32m'
    CYAN=$'\033[36m'
    RED=$'\033[31m'
    YELLOW=$'\033[33m'
    RESET=$'\033[0m'
else
    BOLD=""; DIM=""; GREEN=""; CYAN=""; RED=""; YELLOW=""; RESET=""
fi

step()    { printf "%s==>%s %s\n" "$CYAN" "$RESET" "$*"; }
ok()      { printf "%s✓%s %s\n" "$GREEN" "$RESET" "$*"; }
warn()    { printf "%s!%s %s\n" "$YELLOW" "$RESET" "$*" >&2; }
fail()    { printf "%s✗%s %s\n" "$RED" "$RESET" "$*" >&2; exit 1; }

banner() {
    printf "\n"
    printf "%s========================================%s\n" "$BOLD" "$RESET"
    printf "%s          Oriyn CLI Installer%s\n"           "$BOLD" "$RESET"
    printf "%s========================================%s\n" "$BOLD" "$RESET"
    printf "\n"
}

tmp=""
tmp_checksums=""
cleanup() { rm -f "$tmp" "$tmp_checksums"; }
trap cleanup EXIT

detect_target() {
    local os arch
    os="$(uname -s)"
    arch="$(uname -m)"

    case "$os" in
        Linux)  os="linux" ;;
        Darwin) os="darwin" ;;
        *) fail "Unsupported OS: $os. Download manually: https://github.com/${REPO}/releases/latest" ;;
    esac

    case "$arch" in
        x86_64|amd64)  arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) fail "Unsupported architecture: $arch" ;;
    esac

    printf "%s-%s" "$os" "$arch"
}

checksum_cmd() {
    case "$(uname -s)" in
        Linux)  echo "sha256sum" ;;
        Darwin) echo "shasum -a 256" ;;
    esac
}

resolve_version() {
    if [ -n "${ORIYN_VERSION:-}" ]; then
        echo "${ORIYN_VERSION#v}"
        return
    fi
    local location
    location="$(curl -sI "https://github.com/${REPO}/releases/latest" | grep -i ^location: | tr -d '\r')"
    echo "$location" | sed 's|.*/v||'
}

path_hint() {
    case ":${PATH}:" in
        *":${BIN_DIR}:"*) return 0 ;;
    esac
    warn "${BIN_DIR} is not in your PATH."
    printf "%s  Add this to your shell profile:%s\n" "$DIM" "$RESET"
    printf "    export PATH=\"%s:\$PATH\"\n\n" "$BIN_DIR"
}

main() {
    banner

    local target version binary_name base_url binary_url checksums_url
    target="$(detect_target)"
    step "Detected platform: ${BOLD}${target}${RESET}"

    step "Fetching latest release..."
    version="$(resolve_version)"
    [ -n "$version" ] || fail "Could not resolve latest version. Try setting ORIYN_VERSION=x.y.z"

    binary_name="${BIN_NAME}-${target}"
    base_url="https://github.com/${REPO}/releases/download/v${version}"
    binary_url="${base_url}/${binary_name}"
    checksums_url="${base_url}/checksums.txt"

    if command -v "$BIN_NAME" &>/dev/null; then
        local current
        current="$("$BIN_NAME" --version 2>/dev/null | awk '{print $NF}' | tr -d '()')"
        if [ -n "$current" ]; then
            step "Installing version: ${BOLD}v${version}${RESET} ${DIM}(upgrading from v${current})${RESET}"
        else
            step "Installing version: ${BOLD}v${version}${RESET}"
        fi
    else
        step "Installing version: ${BOLD}v${version}${RESET}"
    fi

    tmp="$(mktemp)"
    tmp_checksums="$(mktemp)"

    step "Downloading ${binary_name}..."
    curl -fSL --progress-bar -o "$tmp" "$binary_url" \
        || fail "Download failed: $binary_url"
    curl -fsSL -o "$tmp_checksums" "$checksums_url" \
        || fail "Checksum file download failed: $checksums_url"

    step "Verifying checksum..."
    local expected actual
    expected="$(grep "${binary_name}" "$tmp_checksums" | awk '{print $1}')"
    actual="$($(checksum_cmd) "$tmp" | awk '{print $1}')"
    [ -n "$expected" ] || fail "No checksum entry found for ${binary_name}"
    [ "$expected" = "$actual" ] || fail "Checksum mismatch (expected ${expected}, got ${actual})"
    ok "Checksum verified"

    step "Installing binary..."
    mkdir -p "$SHARE_DIR" "$BIN_DIR"
    chmod +x "$tmp"
    mv "$tmp" "${SHARE_DIR}/${BIN_NAME}"
    ok "Installed to ${SHARE_DIR}"

    ln -sf "${SHARE_DIR}/${BIN_NAME}" "${BIN_DIR}/${BIN_NAME}"
    ok "Symlinked to ${BIN_DIR}/${BIN_NAME}"

    printf "\n"
    ok "${BOLD}Installation complete!${RESET}"
    printf "\n"
    path_hint
    printf "Run %s%s --help%s to get started.\n" "$BOLD" "$BIN_NAME" "$RESET"
}

main
