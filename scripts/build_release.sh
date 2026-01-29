#!/bin/bash
# eckWMS Multi-Arch Build Script
set -e

OS=${1:-$(go env GOOS)}
ARCH=${2:-$(go env GOARCH)}
OUTPUT="eckwms"

if [ "$OS" == "windows" ]; then
    OUTPUT="eckwms.exe"
fi

echo "🚀 Starting Build for $OS/$ARCH..."

# 1. Build Frontend (Critical: must be done before Go build for embed)
echo "📦 Building Frontend (SvelteKit)..."
cd web
# Set BASE_PATH from .env if available, or default to /E
PREFIX=$(grep HTTP_PATH_PREFIX ../.env | cut -d '=' -f2)
BASE_PATH=${PREFIX:-/E} npm run build
cd ..

# 2. Build Backend
echo "🔨 Compiling Go binary..."
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
COMMIT_TIME=$(git log -1 --format=%cI 2>/dev/null || echo "unknown")
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS="-s -w \
  -X github.com/xelth-com/eckwmsgo/internal/buildinfo.BuildTime=${BUILD_TIME} \
  -X github.com/xelth-com/eckwmsgo/internal/buildinfo.CommitTime=${COMMIT_TIME} \
  -X github.com/xelth-com/eckwmsgo/internal/buildinfo.CommitHash=${COMMIT_HASH}"
CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH go build -ldflags="$LDFLAGS" -o $OUTPUT ./cmd/api

echo "✅ Build Complete: $OUTPUT ($OS/$ARCH)"
if [ "$OS" == "linux" ] && [ "$ARCH" == "arm64" ]; then
    cp $OUTPUT eckwms-linux-arm64
    # Also copy to eckwmsgo (the name systemd expects)
    mv $OUTPUT eckwmsgo
    chmod +x eckwmsgo
    echo "📦 Saved as eckwms-linux-arm64 + eckwmsgo"
fi
