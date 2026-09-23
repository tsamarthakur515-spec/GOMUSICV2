# GOMUSICV2

Kanha-themed Telegram music bot in Go.

Voice chats use **gogram + ntgcalls**. Start / help / player / queue panels match KanhaMusic (blockquote captions, coloured buttons, incoming-track + now-playing markup). Queues stay in memory — no database.

| thing | version |
| --- | --- |
| go | 1.26.0+ |
| telegram library | gogram v1.7.71 |
| voice calls | ntgcalls v2.2.5 |
| player | ffmpeg + yt-dlp |

## Install Go (if `go version` fails)

```bash
cd /tmp
curl -fsSLO https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
export PATH=/usr/local/go/bin:$PATH
go version
```

Need `gcc`, `ffmpeg`, `yt-dlp` too:

```bash
apt-get update && apt-get install -y build-essential ffmpeg
```

## Run

```bash
cd /root/GOMUSICV2
export PATH=/usr/local/go/bin:$PATH

cp sample.env .env
# fill API_ID API_HASH BOT_TOKEN STRING_SESSION OWNER_ID

go run setup_ntgcalls.go
export CGO_ENABLED=1
go build -o gomusic ./GOMUSIC
pkill -f './gomusic' || true
./gomusic
```
