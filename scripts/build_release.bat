@echo off
REM eckWMS Release Build Script
REM Usage: scripts\build_release.bat

echo 🚀 Starting eckWMS Release Build...

REM 1. Build Frontend
echo 📦 Building Frontend (SvelteKit)...
cd web
call npm install
call npm run build
cd ..

REM 2. Build Backend
echo 🔨 Building Backend (Go)...
go build -o eckwms.exe cmd/api/main.go

echo ✅ Build Complete: eckwms.exe
echo    Run with: .\eckwms.exe
