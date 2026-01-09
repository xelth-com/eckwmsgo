#!/bin/bash
set -e # Exit on error

echo "🚀 Starting eckWMS Release Build..."

# 1. Build Frontend
echo "📦 Building Frontend (SvelteKit)..."
cd web
npm install
npm run build
cd ..

# 2. Build Backend
echo "🔨 Building Backend (Go)..."
go build -o eckwms.exe cmd/api/main.go

echo "✅ Build Complete: eckwms.exe"
echo "   Run with: ./eckwms.exe"
