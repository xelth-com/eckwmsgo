# ⚡ Quick Deploy Reference

## Server Details
- **Host:** antigravity (152.53.15.15)
- **Path:** `/var/www/eckwmsgo`
- **Service:** `eckwmsgo`
- **URL:** https://pda.repair/E

## Deploy Now 🚀

```bash
make deploy-quick
```

## Monitor 📊

```bash
# Check status
make status

# View logs
make logs

# SSH to server
make ssh
```

## Manual Deploy (if needed)

```bash
ssh antigravity
cd /var/www/eckwmsgo
git pull
go mod tidy
go build -ldflags="-s -w" -buildvcs=false -o eckwms cmd/api/main.go
systemctl restart eckwmsgo
systemctl status eckwmsgo
```

## Update .env (AI Key)

```bash
ssh antigravity
nano /var/www/eckwmsgo/.env
# Add: GEMINI_API_KEY=your_key_here
systemctl restart eckwmsgo
```

## Quick Checks

```bash
# Health
curl https://pda.repair/E/health

# Logs (last 50)
ssh antigravity 'journalctl -u eckwmsgo -n 50 --no-pager'

# Service status
ssh antigravity 'systemctl status eckwmsgo --no-pager'
```

## Rollback

```bash
ssh antigravity
cd /var/www/eckwmsgo
git reset --hard HEAD~1
go build -ldflags="-s -w" -buildvcs=false -o eckwms cmd/api/main.go
systemctl restart eckwmsgo
```

---

**See DEPLOY.md for full documentation**
