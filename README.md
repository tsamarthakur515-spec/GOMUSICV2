# GOMUSICV2

Kanha-themed Telegram music bot in Go.

Voice chats use **gogram + ntgcalls**. Start / help / player / queue panels use blockquote captions and coloured buttons. Queues stay in memory — no database.

| thing | version |
| --- | --- |
| go | 1.26.0+ |
| telegram library | gogram |
| voice calls | ntgcalls |
| player | ffmpeg + yt-dlp |

## 1. Server packages

Run as root (no `sudo` needed if you already are root):

```bash
apt-get update
apt-get install -y build-essential ffmpeg fonts-dejavu curl git python3
```

`yt-dlp` (latest):

```bash
curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp
chmod +x /usr/local/bin/yt-dlp
yt-dlp --version
```

## 2. Install Go

```bash
cd /tmp
curl -fsSLO https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
export PATH=/usr/local/go/bin:$PATH
echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
go version
```

## 3. Clone and configure

```bash
cd /root
git clone https://github.com/nikhil390u8o/GOMUSICV2.git
cd /root/GOMUSICV2
cp sample.env .env
```

Edit `.env` and fill:

- `API_ID`
- `API_HASH`
- `BOT_TOKEN`
- `STRING_SESSION`
- `OWNER_ID`

## 4. Build and run

```bash
cd /root/GOMUSICV2
export PATH=/usr/local/go/bin:$PATH
export CGO_ENABLED=1

go run setup_ntgcalls.go
go build -o gomusic ./GOMUSIC
pkill -f './gomusic' || true
./gomusic
```

Keep it running after SSH disconnect:

```bash
cd /root/GOMUSICV2
export PATH=/usr/local/go/bin:$PATH
export CGO_ENABLED=1
nohup ./gomusic > /root/gomusic.log 2>&1 &
```

## 5. Update later

```bash
cd /root/GOMUSICV2
git pull
export PATH=/usr/local/go/bin:$PATH
export CGO_ENABLED=1
pkill -f gomusic || true
go build -o gomusic ./GOMUSIC
./gomusic
```

## Notes

- Thumbnails are drawn by ffmpeg (`GOMUSIC/thumb.go`). Need `ffmpeg` + `fonts-dejavu`.
- Hide thumbs in a chat with `/nothumb`.
- First build can take a few minutes while Go downloads modules.
