# GOMUSICV2

Private Go Telegram VC music bot. Same features as GOMUSIC.
Layout like PANDA-MUSICV4: thin root + one code folder.

```
main files at root
GOMUSIC/     all bot code
ntgcalls/    C bindings (run setup once)
```

## Run

```bash
cp sample.env .env
# fill API_ID API_HASH BOT_TOKEN STRING_SESSION OWNER_ID

go run setup_ntgcalls.go
export CGO_ENABLED=1
go build -o gomusic ./GOMUSIC
./gomusic
```

Needs Go 1.26+, gcc, ffmpeg, yt-dlp.
