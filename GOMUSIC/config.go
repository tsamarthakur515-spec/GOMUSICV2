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

func loadLoggerID() int64 {
	for _, k := range []string{"LOGGER_ID", "LOG_GROUP_ID"} {
		v := strings.TrimSpace(os.Getenv(k))
		if v == "" {
			continue
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil && n != 0 {
			return n
		}
	}
	return 0
}

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
	LoggerID = loadLoggerID()
	PingImgURL = envOr("PING_IMG_URL", "https://files.catbox.moe/ddzvc0.jpg")
	SessionName = envOr("SESSION_NAME", "ShizuMusic")
	Port, _ = strconv.Atoi(envOr("PORT", "10000"))
	StartPhotos = splitCSV(envOr("START_PHOTOS", "https://files.catbox.moe/jgt2vm.png"))
	MaxDurationSeconds, _ = strconv.Atoi(envOr("MAX_DURATION_SECONDS", "1800"))
	QueueLimit, _ = strconv.Atoi(envOr("QUEUE_LIMIT", "20"))
	Cooldown, _ = strconv.Atoi(envOr("COOLDOWN", "10"))
	ShrutiAPIURL = strings.TrimRight(envOr("SHRUTI_API_URL", "https://aruyt.up.railway.app"), "/")
	ShrutiAPIKey = envOr("SHRUTI_API_KEY", "")
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

func splitCSV(s string) string {
	return strings.TrimSpace(s)
}
