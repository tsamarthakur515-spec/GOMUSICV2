#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

export PATH=/usr/local/go/bin:$PATH
export CGO_ENABLED=1

echo "==> GOMUSICV2 setup"

if [ "$(id -u)" -eq 0 ]; then
  SUDO=""
else
  SUDO="sudo"
fi

echo "==> packages"
$SUDO apt update -y
$SUDO apt install -y build-essential gcc g++ make ffmpeg fonts-dejavu-core fonts-dejavu curl git python3 wget unzip ca-certificates

if ! command -v go >/dev/null 2>&1; then
  echo "==> installing Go 1.26.0"
  cd /tmp
  curl -fsSLO https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
  $SUDO rm -rf /usr/local/go
  $SUDO tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
  cd "$ROOT"
fi
export PATH=/usr/local/go/bin:$PATH
go version

if ! command -v yt-dlp >/dev/null 2>&1; then
  echo "==> installing yt-dlp"
  $SUDO curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp
  $SUDO chmod +x /usr/local/bin/yt-dlp
fi

if [ ! -f .env ]; then
  cp sample.env .env
  echo
  echo "Created .env from sample.env"
  echo "Fill API_ID API_HASH BOT_TOKEN STRING_SESSION OWNER_ID then run:"
  echo "  bash setup.sh"
  exit 1
fi

need() {
  grep -E "^$1=.+" .env >/dev/null || {
    echo "missing $1 in .env"
    exit 1
  }
}
need API_ID
need API_HASH
need BOT_TOKEN
need STRING_SESSION
need OWNER_ID

echo "==> stop old bot (same session cannot run on two IPs)"
pkill -f './gomusic' 2>/dev/null || true
pkill -f '/gomusic' 2>/dev/null || true
sleep 1

echo "==> ntgcalls"
go run setup_ntgcalls.go

echo "==> patch illegal rune if present"
python3 - <<'PY'
from pathlib import Path
p = Path("GOMUSIC/thumb.go")
if not p.exists():
    raise SystemExit(0)
t = p.read_text(encoding="utf-8")
n = t.replace("case 'x', '\u1d07x':", "case 'x':")
n = n.replace("case 'x', '\u1d07x':", "case 'x':")
# raw two-char literal used in older copies
n = n.replace("case 'x', '\u1d07x':", "case 'x':")
if "'\u1d07x'" in n or "case 'x'," in n:
    n = n.replace(", '\u1d07x'", "")
if n != t:
    p.write_text(n, encoding="utf-8")
    print("thumb.go patched")
else:
    print("thumb.go ok")
PY

echo "==> build"
go build -o gomusic ./GOMUSIC

echo "==> start"
nohup ./gomusic > /root/gomusic.log 2>&1 &
sleep 2
echo "log: /root/gomusic.log"
tail -n 30 /root/gomusic.log || true
echo
echo "Done. Follow logs: tail -f /root/gomusic.log"
