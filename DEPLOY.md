# 🚀 eckWMS Deployment Guide

## Server Information
- **Server:** antigravity (152.53.15.15)
- **Project Path:** `/var/www/eckwmsgo`
- **Service:** `eckwmsgo` (systemd)
- **URL:** https://pda.repair/E
- **SSH Key:** `~/.ssh/netcup`

## Quick Deploy to Production

### Method 1: One-line deploy via SSH ⚡ (Recommended)
```bash
ssh antigravity 'cd /var/www/eckwmsgo && git pull && go mod tidy && go build -ldflags="-s -w" -buildvcs=false -o eckwms cmd/api/main.go && systemctl restart eckwmsgo && systemctl status eckwmsgo --no-pager'
```

### Method 2: Using deploy script
```bash
# From local machine
cat deploy.sh | ssh antigravity 'bash -s'
```

### Method 3: Using Makefile
```bash
# Quick deploy
make deploy-quick

# Or full deploy with checks
make deploy
```

### Method 4: Manual steps on server
```bash
# SSH into server (as root)
ssh antigravity

# Navigate to project
cd /var/www/eckwmsgo

# Pull latest code
git pull origin master

# Update dependencies
go mod tidy

# Build optimized binary
go build -ldflags="-s -w" -buildvcs=false -o eckwms cmd/api/main.go

# Restart service
systemctl restart eckwmsgo

# Check status
systemctl status eckwmsgo --no-pager
```

## Post-Deployment Checks

### 1. Check Service Status
```bash
# On server
systemctl status eckwmsgo

# Or from local machine
ssh antigravity 'systemctl status eckwmsgo --no-pager'
```

### 2. View Live Logs
```bash
# On server
journalctl -u eckwmsgo -f

# Or from local machine
ssh antigravity 'journalctl -u eckwmsgo -f'
```

### 3. Check Last 50 Lines
```bash
journalctl -u eckwmsgo -n 50 --no-pager
```

### 4. Test API Health
```bash
# From anywhere
curl https://pda.repair/E/health

# Expected response:
# {"status":"ok","server":"local"}
```

### 5. Test AI Integration (if GEMINI_API_KEY configured)
```bash
# Check logs for AI initialization
ssh antigravity 'journalctl -u eckwmsgo | grep -i gemini'

# Should show:
# 🧠 Initializing Gemini AI Client (Official SDK)...
# ✅ AI Client initialized (model: gemini-3-flash-preview)
```

## Rollback

If deployment fails:

```bash
# SSH to server
ssh antigravity

# Stop the service
systemctl stop eckwmsgo

# Navigate to project
cd /var/www/eckwmsgo

# Revert to previous commit
git reset --hard HEAD~1

# Rebuild
go build -ldflags="-s -w" -buildvcs=false -o eckwms cmd/api/main.go

# Restart
systemctl start eckwmsgo

# Check status
systemctl status eckwmsgo
```

## Environment Variables

Production `.env` location: `/var/www/eckwmsgo/.env`

### Edit .env on server:
```bash
ssh antigravity
cd /var/www/eckwmsgo
nano .env
```

### Key variables for AI features:
```bash
GEMINI_API_KEY=your_actual_api_key
GEMINI_MODEL=gemini-3-flash-preview
GEMINI_MODEL_FALLBACK=gemini-2.5-flash
```

### After changing `.env`:
```bash
systemctl restart eckwmsgo
```

### Quick .env check from local machine:
```bash
ssh antigravity 'cat /var/www/eckwmsgo/.env | grep GEMINI'
```

## Useful Service Commands

All commands assume you're logged in as root via `ssh antigravity`

```bash
# Start service
systemctl start eckwmsgo

# Stop service
systemctl stop eckwmsgo

# Restart service
systemctl restart eckwmsgo

# Check status
systemctl status eckwmsgo

# Enable auto-start on boot
systemctl enable eckwmsgo

# Disable auto-start
systemctl disable eckwmsgo

# View service configuration
systemctl cat eckwmsgo

# Reload systemd after config changes
systemctl daemon-reload
```

### From local machine (one-liners):
```bash
# Restart remotely
ssh antigravity 'systemctl restart eckwmsgo'

# Check status remotely
ssh antigravity 'systemctl status eckwmsgo --no-pager'

# View logs remotely
ssh antigravity 'journalctl -u eckwmsgo -n 100 --no-pager'
```

## Build Flags Explained

- `-ldflags="-s -w"` - Strip debug symbols (reduces binary size ~30%)
- `-buildvcs=false` - Disable VCS stamping (faster builds)
- `-o eckwms` - Output filename

## Troubleshooting

### Service won't start
```bash
# SSH to server
ssh antigravity

# Check for errors in logs
journalctl -u eckwmsgo -n 100 --no-pager

# Check if port is already in use
lsof -i :3210

# Verify binary exists and has correct permissions
ls -l /var/www/eckwmsgo/eckwms

# Try running manually for debug
cd /var/www/eckwmsgo
./eckwms
```

### Database connection issues
```bash
# Check if PostgreSQL is running
systemctl status postgresql

# Test connection
psql -h localhost -U wms_user -d eckwms

# Check database config in .env
cat /var/www/eckwmsgo/.env | grep PG_
```

### AI features not working
```bash
# Check if API key is set
grep GEMINI_API_KEY /var/www/eckwmsgo/.env

# Check initialization logs
journalctl -u eckwmsgo | grep -i "gemini\|ai"

# Test API key manually
curl -X POST https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:generateContent?key=YOUR_KEY \
  -H 'Content-Type: application/json' \
  -d '{"contents":[{"parts":[{"text":"test"}]}]}'
```

### Build fails
```bash
# Check Go version (need 1.21+)
go version

# Clean and rebuild
cd /var/www/eckwmsgo
rm -f eckwms
go clean -cache
go mod tidy
go build -ldflags="-s -w" -buildvcs=false -o eckwms cmd/api/main.go
```

## Performance Monitoring

### On server:
```bash
# SSH to server
ssh antigravity

# CPU/Memory usage of eckwms process
top -p $(pgrep -f eckwms)

# Or use htop for better visualization
htop -p $(pgrep -f eckwms)

# Network connections
netstat -tulpn | grep eckwms
# Or
ss -tulpn | grep eckwms

# Active database connections
psql -U wms_user -d eckwms -c "SELECT count(*) FROM pg_stat_activity;"

# Disk usage
df -h /var/www/eckwmsgo

# Binary size
ls -lh /var/www/eckwmsgo/eckwms
```

### Remote monitoring from local machine:
```bash
# Quick status check
ssh antigravity 'systemctl status eckwmsgo --no-pager | head -20'

# Memory usage
ssh antigravity 'ps aux | grep eckwms | grep -v grep'

# Check database size
ssh antigravity 'psql -U wms_user -d eckwms -c "SELECT pg_size_pretty(pg_database_size(current_database()));"'
```
