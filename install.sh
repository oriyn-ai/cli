#!/usr/bin/env bash
# Oriyn CLI installer.
#
# Tries `bun add -g oriyn` first; falls back to a precompiled binary from the
# latest GitHub release if Bun is not installed.
#
#   curl -fsSL https://oriyn.ai/install.sh | bash
#
# Override knobs:
#   ORIYN_VERSION   — pin a specific tag (default: latest release)
#   ORIYN_INSTALL_DIR — install dir for the binary fallback (default: $HOME/.local/bin)
#   ORIYN_REPO      — github repo (default: oriyn-ai/cli)

set -euo pipefail

REPO="${ORIYN_REPO:-oriyn-ai/cli}"
INSTALL_DIR="${ORIYN_INSTALL_DIR:-$HOME/.local/bin}"

log() { printf '%s\n' "$*" >&2; }
err() { log "error: $*"; exit 1; }

if command -v bun >/dev/null 2>&1; then
  log "bun detected — installing via 'bun add -g oriyn'"
  if [[ -n "${ORIYN_VERSION:-}" ]]; then
    bun add -g "oriyn@${ORIYN_VERSION#v}"
  else
    bun add -g oriyn@latest
  fi
  log "✓ installed. Run: oriyn auth login"
  exit 0
fi

# --- Binary fallback ---

uname_s=$(uname -s)
uname_m=$(uname -m)
case "$uname_s" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) err "Unsupported OS: $uname_s. Install Bun first: https://bun.com" ;;
esac
case "$uname_m" in
  arm64|aarch64) arch=arm64 ;;
  x86_64|amd64) arch=x64 ;;
  *) err "Unsupported architecture: $uname_m" ;;
esac

mkdir -p "$INSTALL_DIR"
asset="oriyn-${os}-${arch}"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

if [[ -n "${ORIYN_VERSION:-}" ]]; then
  tag="${ORIYN_VERSION}"
else
  tag=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)
  [[ -n "$tag" ]] || err "Could not resolve latest release tag"
fi
log "Downloading ${asset} ${tag}…"
url="https://github.com/${REPO}/releases/download/${tag}/${asset}"
curl -fSL "$url" -o "$tmp/oriyn"
chmod +x "$tmp/oriyn"
mv "$tmp/oriyn" "$INSTALL_DIR/oriyn"
log "✓ installed to ${INSTALL_DIR}/oriyn"
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) log "Add ${INSTALL_DIR} to your PATH to use 'oriyn' globally." ;;
esac
log "Run: oriyn auth login"
