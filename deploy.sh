#!/bin/bash
# eckWMS Production Deployment Script
# Server: antigravity (152.53.15.15)
# Project: /var/www/eckwmsgo
# Usage: cat deploy.sh | ssh antigravity 'bash -s'

set -e  # Exit on error

echo "🚀 eckWMS Deployment Started"
echo "================================"
echo "Server: antigravity (152.53.15.15)"
echo "Project: eckwmsgo"
echo ""

# Navigate to project directory
PROJECT_PATH="/var/www/eckwmsgo"
cd "$PROJECT_PATH" || { echo "❌ Project directory not found: $PROJECT_PATH"; exit 1; }

echo "📂 Current directory: $(pwd)"
echo ""

# 1. Pull latest code
echo "📥 Pulling latest code from master..."
git pull origin master || { echo "❌ Git pull failed"; exit 1; }

# 2. Update dependencies
echo "📦 Updating Go dependencies..."
go mod tidy || { echo "❌ Go mod tidy failed"; exit 1; }

# 3. Build optimized binary
echo "🔨 Building production binary..."
go build -ldflags="-s -w" -buildvcs=false -o eckwms cmd/api/main.go || { echo "❌ Build failed"; exit 1; }

# Check if build succeeded
if [ ! -f eckwms ]; then
    echo "❌ Build failed - binary not created"
    exit 1
fi

# Display binary info
echo "✅ Build complete:"
ls -lh eckwms
echo ""

# 4. Restart production service
echo "🔄 Restarting eckwmsgo service..."
systemctl restart eckwmsgo || { echo "❌ Service restart failed"; exit 1; }

# Wait for service to start
echo "⏳ Waiting for service to start..."
sleep 3

# 5. Check service status
echo "📊 Service Status:"
systemctl status eckwmsgo --no-pager -l | head -20

# 6. Check if service is active
if systemctl is-active --quiet eckwmsgo; then
    echo ""
    echo "✅ Deployment successful!"
    echo "🌐 Service is running at https://pda.repair/E"
    echo ""
    echo "💡 Quick checks:"
    echo "  Health: curl https://pda.repair/E/health"
    echo "  Logs:   journalctl -u eckwmsgo -f"
else
    echo ""
    echo "❌ Service failed to start!"
    echo "📋 Recent logs:"
    journalctl -u eckwmsgo -n 50 --no-pager
    exit 1
fi
