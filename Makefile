# eckWMS Go - Master Control File
.PHONY: setup build build-arm64 build-prod dev clean deploy deploy-quick logs status ssh help

help:
	@echo "eckWMS Go Management Commands:"
	@echo ""
	@echo "🔨 Build:"
	@echo "  setup          - Install all dependencies (Go, NPM, Playwright)"
	@echo "  build          - Build production binary for current OS"
	@echo "  build-prod     - Build optimized production binary (stripped)"
	@echo "  build-arm64    - Cross-compile for Linux ARM64 (TSD/Scanner)"
	@echo ""
	@echo "🚀 Deploy:"
	@echo "  deploy         - Deploy to production server (antigravity)"
	@echo "  deploy-quick   - Quick one-line deploy to production"
	@echo ""
	@echo "🔍 Monitor:"
	@echo "  status         - Check production service status"
	@echo "  logs           - View production logs (live)"
	@echo "  ssh            - SSH to production server"
	@echo ""
	@echo "💻 Development:"
	@echo "  dev            - Run backend in development mode"
	@echo "  clean          - Remove build artifacts"

setup:
	@echo "🚀 Starting full setup..."
	bash scripts/setup.sh --with-playwright

build:
	@echo "🔨 Building for local system..."
	bash scripts/build_release.sh

build-arm64:
	@echo "📦 Cross-compiling for Linux ARM64..."
	bash scripts/build_release.sh linux arm64

dev:
	@echo "🏃 Starting dev server..."
	go run cmd/api/main.go

build-prod:
	@echo "🔨 Building optimized production binary..."
	go build -ldflags="-s -w -X github.com/xelth-com/eckwmsgo/internal/buildinfo.BuildTime=$$(date -u '+%Y-%m-%dT%H:%M:%SZ') -X github.com/xelth-com/eckwmsgo/internal/buildinfo.CommitTime=$$(git log -1 --format=%cI) -X github.com/xelth-com/eckwmsgo/internal/buildinfo.CommitHash=$$(git rev-parse --short HEAD)" -o eckwms cmd/api/main.go
	@ls -lh eckwms

deploy:
	@echo "🚀 Deploying to production server..."
	cat deploy.sh | ssh antigravity 'bash -s'

deploy-quick:
	@echo "⚡ Quick deploy to production (antigravity)..."
	ssh antigravity 'cd /var/www/eckwmsgo && git pull && go mod tidy && go build -ldflags="-s -w -X github.com/xelth-com/eckwmsgo/internal/buildinfo.BuildTime=$$(date -u +%Y-%m-%dT%H:%M:%SZ) -X github.com/xelth-com/eckwmsgo/internal/buildinfo.CommitTime=$$(git log -1 --format=%cI) -X github.com/xelth-com/eckwmsgo/internal/buildinfo.CommitHash=$$(git rev-parse --short HEAD)" -buildvcs=false -o eckwms cmd/api/main.go && systemctl restart eckwmsgo && systemctl status eckwmsgo --no-pager'

status:
	@echo "📊 Checking production service status..."
	@ssh antigravity 'systemctl status eckwmsgo --no-pager'

logs:
	@echo "📋 Viewing production logs (Ctrl+C to exit)..."
	@ssh antigravity 'journalctl -u eckwmsgo -f'

ssh:
	@echo "🔐 Connecting to production server..."
	@ssh antigravity

clean:
	@echo "🧹 Cleaning up..."
	rm -f eckwms eckwms.exe eckwms-linux-arm64
