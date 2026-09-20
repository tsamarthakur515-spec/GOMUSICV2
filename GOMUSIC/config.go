package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

var (
	APIID              int
	APIHash            string
	BotToken           string
	StringSession      string
	OwnerID            int64
	BotName            string
	BotLink            string
	UpdatesChannel     string
	SupportGroup       string
	LoggerID           int64
	PingImgURL         string
	SessionName        string
	Port               int
	StartPhotos        []string
	MaxDurationSeconds int
	QueueLimit         int
	Cooldown           int
	ShrutiAPIURL       string
	ShrutiAPIKey       string
)

func loadConfig() error {
	_ = godotenv.Load()
	var err error
	APIID, err = strconv.Atoi(mustEnv("API_ID"))
	if err != nil {
		return err
	}
	APIHash = mustEnv("API_HASH")
	BotToken = mustEnv("BOT_TOKEN")
	StringSession = mustEnv("STRING_SESSION")
	OwnerID, err = strconv.ParseInt(mustEnv("OWNER_ID"), 10, 64)
	if err != nil {
		return err
	}
	BotName = envOr("BOT_NAME", "Shizu Music")
	BotLink = envOr("BOT_LINK", "https://t.me/ARU_xOPUSERBOT")
	UpdatesChannel = envOr("UPDATES_CHANNEL", "https://t.me/sxypndu")
	SupportGroup = envOr("SUPPORT_GROUP", "https://t.me/crzy_soul")
	LoggerID, _ = strconv.ParseInt(envOr("LOGGER_ID", "0"), 10, 64)
	PingImgURL = envOr("PING_IMG_URL", "https://files.catbox.moe/ddzvc0.jpg")
	SessionName = envOr("SESSION_NAME", "ShizuMusic")
	Port, _ = strconv.Atoi(envOr("PORT", "10000"))
	StartPhotos = splitCSV(envOr("START_PHOTOS", "https://files.catbox.moe/jgt2vm.png"))
	MaxDurationSeconds, _ = strconv.Atoi(envOr("MAX_DURATION_SECONDS", "1800"))
	QueueLimit, _ = strconv.Atoi(envOr("QUEUE_LIMIT", "20"))
	Cooldown, _ = strconv.Atoi(envOr("COOLDOWN", "10"))
	ShrutiAPIURL = strings.TrimRight(envOr("SHRUTI_API_URL", "https://aruyt.up.railway.app"), "/")
	ShrutiAPIKey = envOr("SHRUTI_API_KEY", "YUKI-zi4hcOkYs0tBIAX9QzDc9iTn")
	return nil
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		panic("missing required env: " + k)
	}
	return v
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"https://files.catbox.moe/jgt2vm.png"}
	}
	return out
}
