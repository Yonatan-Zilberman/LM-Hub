#!/usr/bin/env bash

set -euo pipefail

REPO="Yonatan-Zilberman/LM-Hub"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
SOURCE_INSTALL=false

usage() {
    cat <<EOF
Usage: install.sh [OPTIONS]

Install LMH (LM-Hub) to ${INSTALL_DIR}.

Options:
  --source       Build from local source instead of downloading a release
  --dir PATH     Install directory (default: ~/.local/bin)
  -h, --help     Show this help message

Examples:
  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | sh
  ./install.sh --source
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --source)
            SOURCE_INSTALL=true
            shift
            ;;
        --dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            usage
            exit 1
            ;;
    esac
done

detect_platform() {
    local os arch
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)
            echo "Unsupported architecture: $arch" >&2
            exit 1
            ;;
    esac
    case "$os" in
        darwin|linux) ;;
        *)
            echo "Unsupported OS: $os (use --source to build locally)" >&2
            exit 1
            ;;
    esac
    echo "${os} ${arch}"
}

install_from_source() {
    if ! command -v go &> /dev/null; then
        echo "Go is required for --source installs. Install Go 1.22+ from https://go.dev/dl/" >&2
        exit 1
    fi
    echo "Building lmh from source..."
    go build -o lmh ./cmd/lmh
    mkdir -p "$INSTALL_DIR"
    cp lmh "$INSTALL_DIR/lmh"
    chmod +x "$INSTALL_DIR/lmh"
    ln -sf "$INSTALL_DIR/lmh" "$INSTALL_DIR/lmhub"
    rm -f lmh
}

install_from_release() {
    read -r OS ARCH <<< "$(detect_platform)"

    echo "Fetching latest release for ${OS}/${ARCH}..."
    local api_url="https://api.github.com/repos/${REPO}/releases/latest"
    local release_json
    release_json="$(curl -fsSL "$api_url")"

    local version
    version="$(echo "$release_json" | grep '"tag_name":' | head -1 | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')"
    if [[ -z "$version" ]]; then
        echo "Could not determine latest release version." >&2
        exit 1
    fi

    local archive="LM-Hub_${version#v}_${OS}_${ARCH}.tar.gz"
    if [[ "$OS" == "windows" ]]; then
        archive="LM-Hub_${version#v}_${OS}_${ARCH}.zip"
    fi

    local download_url="https://github.com/${REPO}/releases/download/${version}/${archive}"
    local checksums_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"
    local tmpdir
    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT

    echo "Downloading ${download_url}..."
    curl -fsSL -o "${tmpdir}/${archive}" "$download_url"

    if curl -fsSL -o "${tmpdir}/checksums.txt" "$checksums_url" 2>/dev/null; then
        echo "Verifying checksum..."
        (
            cd "$tmpdir"
            if command -v sha256sum &> /dev/null; then
                grep " ${archive}\$" checksums.txt | sha256sum -c -
            else
                grep " ${archive}\$" checksums.txt | shasum -a 256 -c -
            fi
        )
    else
        echo "Warning: checksums.txt not found; skipping verification."
    fi

    echo "Extracting archive..."
    if [[ "$archive" == *.zip ]]; then
        unzip -q "${tmpdir}/${archive}" -d "$tmpdir"
    else
        tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"
    fi

    mkdir -p "$INSTALL_DIR"
    if [[ -f "${tmpdir}/lmh" ]]; then
        cp "${tmpdir}/lmh" "$INSTALL_DIR/lmh"
    elif [[ -f "${tmpdir}/lmh.exe" ]]; then
        cp "${tmpdir}/lmh.exe" "$INSTALL_DIR/lmh.exe"
    else
        local bin
        bin="$(find "$tmpdir" -maxdepth 2 -type f \( -name lmh -o -name lmh.exe -o -name lmhub -o -name lmhub.exe \) | head -1)"
        if [[ -z "$bin" ]]; then
            echo "Could not find lmh binary in release archive." >&2
            exit 1
        fi
        cp "$bin" "$INSTALL_DIR/$(basename "$bin")"
    fi

    if [[ -f "$INSTALL_DIR/lmh" ]]; then
        chmod +x "$INSTALL_DIR/lmh"
        ln -sf "$INSTALL_DIR/lmh" "$INSTALL_DIR/lmhub"
    fi
}

echo "=================================================="
echo "Installing LMH (LM-Hub)"
echo "=================================================="

if $SOURCE_INSTALL; then
    install_from_source
else
    if ! install_from_release; then
        echo "Release install failed; falling back to source build..."
        install_from_source
    fi
fi

echo "=================================================="
echo "Installation complete!"
echo "=================================================="
echo "Installed to: ${INSTALL_DIR}/lmh"
echo "Symlink:      ${INSTALL_DIR}/lmhub"
echo ""
echo "Ensure ${INSTALL_DIR} is in your PATH."
if command -v lmh &> /dev/null; then
    lmh --version
else
    echo "Run: ${INSTALL_DIR}/lmh --version"
fi
echo "=================================================="
