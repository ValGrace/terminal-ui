#!/bin/bash
# Build script for Command History Tracker
# Supports cross-platform builds for Windows, macOS, and Linux

set -e

# Default values
VERSION="${VERSION:-0.1.0}"
GIT_COMMIT="${GIT_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'dev')}"
BUILD_DATE="${BUILD_DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"
OUT_DIR="dist"

# Parse command line arguments
BUILD_ALL=false
BUILD_WINDOWS=false
BUILD_LINUX=false
BUILD_MACOS=false
CLEAN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --all)
            BUILD_ALL=true
            shift
            ;;
        --windows)
            BUILD_WINDOWS=true
            shift
            ;;
        --linux)
            BUILD_LINUX=true
            shift
            ;;
        --macos)
            BUILD_MACOS=true
            shift
            ;;
        --clean)
            CLEAN=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Clean build directory
if [ "$CLEAN" = true ]; then
    echo "Cleaning build directory..."
    rm -rf "$OUT_DIR"
    echo "✓ Clean complete"
    exit 0
fi

# Create output directory
mkdir -p "$OUT_DIR"

# Build flags
LDFLAGS="-X command-history-tracker/internal/version.Version=$VERSION \
         -X command-history-tracker/internal/version.GitCommit=$GIT_COMMIT \
         -X command-history-tracker/internal/version.BuildDate=$BUILD_DATE"

echo "Building Command History Tracker v$VERSION"
echo "Git Commit: $GIT_COMMIT"
echo "Build Date: $BUILD_DATE"
echo ""

# Build function
build_binary() {
    local os=$1
    local arch=$2
    local output=$3
    
    echo "Building for $os/$arch..."
    
    GOOS=$os GOARCH=$arch CGO_ENABLED=1 go build \
        -ldflags "$LDFLAGS" \
        -o "$OUT_DIR/$output" \
        ./cmd/tracker
    
    if [ $? -eq 0 ]; then
        local size=$(du -h "$OUT_DIR/$output" | cut -f1)
        echo "✓ Built $output ($size)"
    else
        echo "✗ Failed to build $output"
        exit 1
    fi
}

# Determine what to build
if [ "$BUILD_ALL" = true ] || ([ "$BUILD_WINDOWS" = false ] && [ "$BUILD_LINUX" = false ] && [ "$BUILD_MACOS" = false ]); then
    # Build all platforms by default
    build_binary "windows" "amd64" "tracker-windows-amd64.exe"
    build_binary "windows" "arm64" "tracker-windows-arm64.exe"
    build_binary "linux" "amd64" "tracker-linux-amd64"
    build_binary "linux" "arm64" "tracker-linux-arm64"
    build_binary "darwin" "amd64" "tracker-darwin-amd64"
    build_binary "darwin" "arm64" "tracker-darwin-arm64"
else
    if [ "$BUILD_WINDOWS" = true ]; then
        build_binary "windows" "amd64" "tracker-windows-amd64.exe"
        build_binary "windows" "arm64" "tracker-windows-arm64.exe"
    fi
    
    if [ "$BUILD_LINUX" = true ]; then
        build_binary "linux" "amd64" "tracker-linux-amd64"
        build_binary "linux" "arm64" "tracker-linux-arm64"
    fi
    
    if [ "$BUILD_MACOS" = true ]; then
        build_binary "darwin" "amd64" "tracker-darwin-amd64"
        build_binary "darwin" "arm64" "tracker-darwin-arm64"
    fi
fi

echo ""
echo "Build complete! Binaries are in the '$OUT_DIR' directory."
