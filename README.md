# GOMUSICV2

Kanha-themed Telegram music bot in Go.

Voice chats use **gogram + ntgcalls**. Start / help / player / queue panels use blockquote captions and coloured buttons. Queues stay in memory — no database.

| thing | version |
| --- | --- |
| go | 1.26.0+ |
| telegram library | gogram |
| voice calls | ntgcalls |
| player | ffmpeg + yt-dlp |

## One command setup

On a fresh Ubuntu/Debian VPS as root:

```bash
cd /root/GOMUSICV2
cp sample.env .env
nano .env
bash setup.sh
```

`setup.sh` installs packages, Go, yt-dlp, ntgcalls, builds `gomusic`, stops any old process, and starts the bot.

Fill `.env` first:

- `API_ID`
- `API_HASH`
- `BOT_TOKEN`
- `STRING_SESSION`
- `OWNER_ID`

Logs:

```bash
tail -f /root/gomusic.log
```

## Manual commands (if you do not use setup.sh)

```bash
apt-get update
apt-get install -y build-essential ffmpeg fonts-dejavu curl git python3

curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp
chmod +x /usr/local/bin/yt-dlp

cd /tmp
curl -fsSLO https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
export PATH=/usr/local/go/bin:$PATH

cd /root/GOMUSICV2
cp -n sample.env .env
export CGO_ENABLED=1
go run setup_ntgcalls.go
go build -o gomusic ./GOMUSIC
pkill -f './gomusic' || true
nohup ./gomusic > /root/gomusic.log 2>&1 &
```

## Update later

```bash
cd /root/GOMUSICV2
git pull
bash setup.sh
```

Private repo pull needs a GitHub token, not a password:

```bash
git pull https://USERNAME:TOKEN@github.com/nikhil390u8o/GOMUSICV2.git
```

## AUTH_KEY_DUPLICATED (code 406)

```
assistant connect: AUTH_KEY_DUPLICATED
The authorization key was used under two different IP addresses simultaneously
```

`STRING_SESSION` is the assistant user account. Telegram allows that key from **one IP at a time**.

It happens when the same session is running in two places:

- old Railway / old VPS still online
- two `./gomusic` processes
- phone + VPS both using the same userbot session from different networks at once

Fix:

1. Stop every old copy.

```bash
pkill -f gomusic || true
```

Also stop / delete the old Railway service if it is still deployed.

2. Make a **new** Pyrogram string session for the assistant account.
3. Put only that new value in `.env` as `STRING_SESSION`.
4. Run the bot on **one** server only.

```bash
cd /root/GOMUSICV2
bash setup.sh
```

Do not start the bot on Railway and this VPS together with the same session.

## Notes

- Thumbnails are drawn by ffmpeg (`GOMUSIC/thumb.go`). Need `ffmpeg` + DejaVu fonts.
- Hide thumbs in a chat with `/nothumb`.
- First build can take a few minutes while Go downloads modules.
- `cd \~/GOMUSICV2` is wrong. Use `cd ~/GOMUSICV2` or `cd /root/GOMUSICV2`.
