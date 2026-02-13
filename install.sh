#!/bin/sh
# Ailloy Install Script
# Usage: curl -fsSL https://raw.githubusercontent.com/nimble-giant/ailloy/main/install.sh | bash
#
# Environment variables:
#   AILLOY_VERSION     - Pin a specific version (e.g., "v0.1.0"). Default: latest.
#   AILLOY_INSTALL_DIR - Override install directory. Default: /usr/local/bin or ~/.local/bin.
set -eu

REPO="nimble-giant/ailloy"
BINARY_NAME="ailloy"

# --- Helpers ---

info() {
    printf '  \033[1;34m%s\033[0m %s\n' "$1" "$2"
}

success() {
    printf '  \033[1;32m%s\033[0m %s\n' "$1" "$2"
}

warn() {
    printf '  \033[1;33m%s\033[0m %s\n' "$1" "$2"
}

error() {
    printf '  \033[1;31m%s\033[0m %s\n' "Error:" "$1" >&2
}

fail() {
    error "$1"
    exit 1
}

# --- Detection ---

detect_os() {
    os="$(uname -s)"
    case "$os" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*)
            fail "Windows is not supported by this installer. Use 'go install github.com/${REPO}/cmd/ailloy@latest' or download binaries from https://github.com/${REPO}/releases"
            ;;
        *) fail "Unsupported operating system: $os" ;;
    esac
}

detect_arch() {
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) fail "Unsupported architecture: $arch" ;;
    esac
}

detect_install_dir() {
    if [ -n "${AILLOY_INSTALL_DIR:-}" ]; then
        echo "$AILLOY_INSTALL_DIR"
        return
    fi

    if [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
    elif command -v sudo >/dev/null 2>&1; then
        echo "/usr/local/bin"
    else
        echo "${HOME}/.local/bin"
    fi
}

needs_sudo() {
    target_dir="$1"
    if [ -w "$target_dir" ]; then
        return 1
    fi
    return 0
}

# --- Version ---

get_latest_version() {
    url="https://api.github.com/repos/${REPO}/releases/latest"

    if command -v curl >/dev/null 2>&1; then
        response=$(curl -fsSL "$url" 2>/dev/null) || fail "Failed to fetch latest release. Check your internet connection or set AILLOY_VERSION."
    elif command -v wget >/dev/null 2>&1; then
        response=$(wget -qO- "$url" 2>/dev/null) || fail "Failed to fetch latest release. Check your internet connection or set AILLOY_VERSION."
    else
        fail "Either curl or wget is required to download ailloy."
    fi

    # Parse tag_name from JSON without jq
    version=$(echo "$response" | grep '"tag_name"' | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/')

    if [ -z "$version" ]; then
        fail "Could not determine latest version. GitHub API may be rate-limited. Set AILLOY_VERSION to specify a version."
    fi

    echo "$version"
}

# --- Download & Verify ---

download_file() {
    url="$1"
    dest="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$dest" "$url" || fail "Failed to download: $url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$dest" "$url" || fail "Failed to download: $url"
    fi
}

verify_checksum() {
    binary_path="$1"
    checksums_path="$2"
    binary_filename="$3"

    expected=$(grep "$binary_filename" "$checksums_path" | awk '{print $1}')
    if [ -z "$expected" ]; then
        warn "Warning:" "Could not find checksum for $binary_filename, skipping verification."
        return 0
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$binary_path" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$binary_path" | awk '{print $1}')
    else
        warn "Warning:" "No SHA256 tool found, skipping checksum verification."
        return 0
    fi

    if [ "$actual" != "$expected" ]; then
        fail "Checksum verification failed. Expected: $expected, Got: $actual"
    fi

    success "Verified" "checksum OK"
}

# --- Main ---

main() {
    printf '\n\033[1;35m  %s  Ailloy Installer\033[0m\n\n' "🦊"

    # Detect platform
    os=$(detect_os)
    arch=$(detect_arch)
    info "Platform:" "${os}/${arch}"

    # Determine version
    if [ -n "${AILLOY_VERSION:-}" ]; then
        version="$AILLOY_VERSION"
        info "Version:" "$version (pinned)"
    else
        info "Fetching:" "latest release..."
        version=$(get_latest_version)
        info "Version:" "$version"
    fi

    # Construct download URL
    binary_filename="${BINARY_NAME}-${os}-${arch}"
    download_url="https://github.com/${REPO}/releases/download/${version}/${binary_filename}"
    checksums_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

    # Create temp directory
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    # Download binary and checksums
    info "Downloading:" "$binary_filename..."
    download_file "$download_url" "${tmp_dir}/${binary_filename}"
    download_file "$checksums_url" "${tmp_dir}/checksums.txt"

    # Verify checksum
    verify_checksum "${tmp_dir}/${binary_filename}" "${tmp_dir}/checksums.txt" "$binary_filename"

    # Determine install directory
    install_dir=$(detect_install_dir)

    # Create install dir if needed
    if [ ! -d "$install_dir" ]; then
        mkdir -p "$install_dir"
    fi

    # Install binary
    chmod +x "${tmp_dir}/${binary_filename}"

    if needs_sudo "$install_dir"; then
        info "Installing:" "${install_dir}/${BINARY_NAME} (with sudo)"
        sudo mv "${tmp_dir}/${binary_filename}" "${install_dir}/${BINARY_NAME}"
    else
        info "Installing:" "${install_dir}/${BINARY_NAME}"
        mv "${tmp_dir}/${binary_filename}" "${install_dir}/${BINARY_NAME}"
    fi

    success "Installed:" "${BINARY_NAME} ${version}"

    # Check if install dir is on PATH
    case ":${PATH}:" in
        *":${install_dir}:"*) ;;
        *)
            echo ""
            warn "Note:" "${install_dir} is not in your PATH."
            info "Add it:" "export PATH=\"${install_dir}:\$PATH\""
            info "Or add:" "the above line to your ~/.bashrc, ~/.zshrc, or shell profile."
            ;;
    esac

    # Success message
    echo ""
    success "Ready!" "Run 'ailloy init' to set up your project."
    echo ""
    info "Quick start:" "ailloy init            # Set up AI templates"
    info "            " "ailloy template list   # View available templates"
    info "            " "ailloy customize       # Configure settings"
    echo ""
}

main
