# eckWMS Go - Master Control File
.PHONY: setup build build-arm64 dev clean help

help:
	@echo "eckWMS Go Management Commands:"
	@echo "  setup          - Install all dependencies (Go, NPM, Playwright)"
	@echo "  build          - Build production binary for current OS"
	@echo "  build-arm64    - Cross-compile for Linux ARM64 (TSD/Scanner)"
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

clean:
	@echo "🧹 Cleaning up..."
	rm -f eckwms eckwms.exe eckwms-linux-arm64
