#!/bin/sh
# Human compiler installer
# Usage: curl -fsSL https://raw.githubusercontent.com/barun-bash/human/main/install.sh | sh
# Or:    VERSION=0.4.0 curl -fsSL ... | sh

set -e

REPO="barun-bash/human"
BINARY_NAME="human"
INSTALL_DIR="/usr/local/bin"

# ── Helpers ──

info() {
  printf '  \033[1;34m→\033[0m %s\n' "$1"
}

success() {
  printf '  \033[1;32m✓\033[0m %s\n' "$1"
}

error() {
  printf '  \033[1;31m✗\033[0m %s\n' "$1" >&2
}

# ── Help ──

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
  cat <<EOF
Human compiler installer

Usage:
  curl -fsSL https://raw.githubusercontent.com/barun-bash/human/main/install.sh | sh

Options:
  VERSION=x.y.z   Install a specific version (default: latest)
  INSTALL_DIR=/p   Install to a custom directory (default: /usr/local/bin)

Examples:
  # Install latest
  curl -fsSL https://raw.githubusercontent.com/barun-bash/human/main/install.sh | sh

  # Install specific version
  VERSION=0.4.0 curl -fsSL https://raw.githubusercontent.com/barun-bash/human/main/install.sh | sh

  # Install to custom directory
  INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/barun-bash/human/main/install.sh | sh
EOF
  exit 0
fi

# ── OS Detection ──

detect_os() {
  os="$(uname -s)"
  case "$os" in
    Darwin)  echo "darwin" ;;
    Linux)   echo "linux" ;;
    MINGW*|MSYS*|CYGWIN*)
      error "Windows is not supported by this installer."
      error "Please download the binary manually from:"
      error "  https://github.com/${REPO}/releases"
      exit 1
      ;;
    *)
      error "Unsupported operating system: $os"
      exit 1
      ;;
  esac
}

# ── Architecture Detection ──

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64)  echo "arm64" ;;
    *)
      error "Unsupported architecture: $arch"
      exit 1
      ;;
  esac
}

# ── Version Detection ──

detect_version() {
  if [ -n "${VERSION:-}" ]; then
    echo "$VERSION"
    return
  fi

  info "Fetching latest version..."

  if command -v curl >/dev/null 2>&1; then
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | \
      grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
  elif command -v wget >/dev/null 2>&1; then
    version=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | \
      grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
  else
    error "Neither curl nor wget found. Please install one of them."
    exit 1
  fi

  if [ -z "$version" ]; then
    error "Could not determine the latest version."
    error "Check your internet connection, or set VERSION manually:"
    error "  VERSION=0.4.0 sh install.sh"
    exit 1
  fi

  echo "$version"
}

# ── Download ──

download() {
  url="$1"
  output="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$output"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$output"
  else
    error "Neither curl nor wget found."
    exit 1
  fi
}

# ── Cleanup ──

TMPDIR=""
cleanup() {
  if [ -n "$TMPDIR" ] && [ -d "$TMPDIR" ]; then
    rm -rf "$TMPDIR"
  fi
}
trap cleanup EXIT

# ── Main ──

main() {
  printf '\n  \033[1mHuman Compiler Installer\033[0m\n\n'

  OS=$(detect_os)
  ARCH=$(detect_arch)
  VERSION=$(detect_version)

  info "Platform: ${OS}/${ARCH}"
  info "Version:  v${VERSION}"

  # Construct download URL
  FILENAME="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/${REPO}/releases/download/v${VERSION}/${FILENAME}"

  # Create temp directory
  TMPDIR=$(mktemp -d)

  # Download
  info "Downloading ${FILENAME}..."
  if ! download "$URL" "${TMPDIR}/${FILENAME}"; then
    error "Download failed."
    error "URL: ${URL}"
    error ""
    error "This could mean:"
    error "  - Version v${VERSION} does not exist"
    error "  - No binary available for ${OS}/${ARCH}"
    error "  - Network connectivity issue"
    error ""
    error "Check available releases at:"
    error "  https://github.com/${REPO}/releases"
    exit 1
  fi

  # Extract
  info "Extracting..."
  tar -xzf "${TMPDIR}/${FILENAME}" -C "${TMPDIR}"

  if [ ! -f "${TMPDIR}/${BINARY_NAME}" ]; then
    error "Binary not found in archive."
    exit 1
  fi

  # Install
  info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."

  if [ -w "${INSTALL_DIR}" ]; then
    mv "${TMPDIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
  else
    info "Permission denied — trying with sudo..."
    sudo mv "${TMPDIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
  fi

  # Verify
  if command -v human >/dev/null 2>&1; then
    installed_version=$(human --version 2>/dev/null || echo "unknown")
    success "Installed: ${installed_version}"
  else
    success "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
    echo ""
    info "Make sure ${INSTALL_DIR} is in your PATH:"
    info "  export PATH=\"\$PATH:${INSTALL_DIR}\""
  fi

  echo ""
  info "Get started:"
  info "  human init my-app"
  info "  cd my-app"
  info "  human build app.human"
  echo ""
}

main
