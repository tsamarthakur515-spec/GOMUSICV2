# GOMUSICV2

Private Go Telegram VC music bot.

```
GOMUSIC/     bot code
ntgcalls/    C bindings (run setup once)
```

## Versions

| Thing | Version |
| --- | --- |
| Go | 1.26.0+ |
| Telegram library | gogram v1.7.10 |
| Voice calls | ntgcalls v2.2.5 |
| Player | ffmpeg + yt-dlp |

Menus (`/start`, Help, About) first use Telegram **Bot API** `sendPhoto` / `editMessageMedia` with `parse_mode=HTML` and button `style`. If that fails, they fall back to gogram `SendMedia` / `EditMessage`.

Blockquote comes from HTML tags in the caption (`<blockquote>` and `<blockquote expandable>`), not from a gogram version bump.
Button colors come from Bot API `style` (`danger` / `primary` / `success`) and gogram `KeyboardButtonStyle`.

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

After pulling new code, always rebuild. The old `./gomusic` binary will keep running old menus until you `go build` again.
