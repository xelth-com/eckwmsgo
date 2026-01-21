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
CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH go build -ldflags="-s -w" -o $OUTPUT ./cmd/api

echo "✅ Build Complete: $OUTPUT ($OS/$ARCH)"
if [ "$OS" == "linux" ] && [ "$ARCH" == "arm64" ]; then
    mv $OUTPUT eckwms-linux-arm64
    echo "📦 Saved as eckwms-linux-arm64"
fi
