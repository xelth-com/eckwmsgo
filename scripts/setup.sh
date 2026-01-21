#!/bin/bash
# eckWMS Portability Setup Script (Linux/Mac)
set -e

echo "🔍 Checking requirements..."
command -v go >/dev/null 2>&1 || { echo >&2 "❌ Go is required but not installed."; exit 1; }
command -v node >/dev/null 2>&1 || { echo >&2 "❌ Node.js is required but not installed."; exit 1; }
command -v npm >/dev/null 2>&1 || { echo >&2 "❌ NPM is required but not installed."; exit 1; }

echo "⚙️ Initializing environment..."
if [ ! -f .env ]; then
    cp .env.example .env
    echo "✅ Created .env from .env.example (Please edit it!)"
else
    echo "ℹ️ .env already exists."
fi

echo "📦 Installing Go dependencies..."
go mod tidy

echo "🌐 Installing Frontend dependencies..."
cd web
npm install
cd ..

echo "🚚 Installing Delivery scripts dependencies..."
cd scripts/delivery
npm install
if [ "$1" == "--with-playwright" ]; then
    echo "🎭 Installing Playwright browsers..."
    npx playwright install chromium
fi
cd ../..

echo "✨ Setup complete! Use scripts/build_release.sh to compile."
