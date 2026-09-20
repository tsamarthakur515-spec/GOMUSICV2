package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

var blockedWords = []string{"blocked"}
var lastCmd = map[int64]time.Time{}

func registerHandlers() {
	Bot.On("message:/start", handleStart)
	Bot.On("message:/help", handleHelp)
	Bot.On("message:/play", handlePlay)
	Bot.On("message:/vplay", handleVPlay)
	Bot.On("message:/pause", handlePause)
	Bot.On("message:/resume", handleResume)
	Bot.On("message:/skip", handleSkip)
	Bot.On("message:/stop", handleStop)
	Bot.On("message:/end", handleStop)
	Bot.On("message:/clear", handleClear)
	Bot.On("message:/reboot", handleReboot)
	Bot.On("message:/ping", handlePing)
	Bot.On("message:/id", handleID)
	Bot.On("message:/autoplay", handleAutoplay)
	Bot.On("message:/speed", handleSpeed)
	Bot.On("message:/speedreset", handleSpeedReset)
	Bot.On("message:/bass", handleBass)
	Bot.On("message:/bassoff", handleBassOff)
	Bot.On("message:/effecton", handleEffectOn)
	Bot.On("message:/effectoff", handleEffectOff)
	Bot.On("message:/effects", handleEffects)
	Bot.On("message:/seek", handleSeek)
	Bot.On("message:/seekback", handleSeekBack)
	Bot.On("message:/gblock", handleGBlock)
	Bot.On("message:/gunblock", handleGUnblock)
	Bot.On("message:/ublock", handleUBlock)
	Bot.On("message:/uunblock", handleUUnblock)
	Bot.On("message:/blocklist", handleBlocklist)
	Bot.On("message:/broadcast", handleBroadcast)
	Bot.On("message:/gcast", handleBroadcast)
	Bot.On("message:/stats", handleStats)
	Bot.On(telegram.OnCallbackQuery, handleCallbackQuery)
}

func cmdArgs(m *telegram.NewMessage) string {
	parts := strings.SplitN(strings.TrimSpace(m.Text()), " ", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func chatIDOf(m *telegram.NewMessage) int64 { return m.ChatID() }

func userIDOf(m *telegram.NewMessage) int64 {
	if m.Sender == nil {
		return 0
	}
	return m.Sender.ID
}

func userNameOf(m *telegram.NewMessage) string {
	if m.Sender == nil {
		return "Unknown"
	}
	return sanitizeDisplayName(m.Sender.FirstName)
}

func mentionOf(m *telegram.NewMessage) string {
	if m.Sender == nil {
		return "Unknown"
	}
	return fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, m.Sender.ID, richEsc(m.Sender.FirstName))
}

func isOwner(id int64) bool { return id == OwnerID || id == 777000 }

func isAuthorized(m *telegram.NewMessage) bool {
	uid := userIDOf(m)
	if isOwner(uid) {
		return true
	}
	if m.IsPrivate() {
		return false
	}
	member, err := Bot.GetChatMember(m.ChatID(), uid)
	if err != nil || member == nil {
		return false
	}
	st := strings.ToLower(fmt.Sprint(member.Status))
	return strings.Contains(st, "admin") || strings.Contains(st, "creator") || strings.Contains(st, "owner")
}

func blocked(m *telegram.NewMessage) bool {
	if isGroupBlocked(m.ChatID()) {
		return true
	}
	return isUserBlockedDB(userIDOf(m))
}

func startHomeKB() telegram.ReplyMarkup {
	return mixedKeyboard([][][2]string{
		{{"Add me to your group", BotLink + "?startgroup=true"}},
		{{"Owner", fmt.Sprintf("tg://user?id=%d", OwnerID)}, {"About", "about_menu"}},
		{{"Support", SupportGroup}, {"Updates", UpdatesChannel}},
		{{"Help and commands", "show_help"}},
	})
}

func aboutKB() telegram.ReplyMarkup {
	return mixedKeyboard([][][2]string{
		{{"Back", "go_back"}},
	})
}

func startCaption(uid int64, name string) string {
	return fmt.Sprintf(
		"hey <a href='tg://user?id=%d'>%s</a>\n\n"+
			"high quality fast music bot.\n"+
			"add me to a group for audio / video vc.\n\n"+
			"use the buttons below.",
		uid, richEsc(name),
	)
}

func aboutCaption() string {
	return richHeading("about", 3) +
		richKVTable([][2]string{
			{"bot", "<code>" + richEsc(BotName) + "</code>"},
			{"version", "<code>GOMUSIC v2</code>"},
			{"language", "<code>Go</code>"},
			{"telegram", "<code>gogram</code>"},
			{"calls", "<code>ntgcalls v2.2.5</code>"},
			{"player", "<code>ffmpeg + yt-dlp</code>"},
			{"runtime", "<code>" + runtime.Version() + "</code>"},
		}) +
		richNote("telegram music bot written in go.\nsupports /play and /vplay in voice chat.")
}

func handleStart(m *telegram.NewMessage) error {
	if blocked(m) {
		return nil
	}
	_, _ = m.Delete()
	uid := userIDOf(m)
	name := userNameOf(m)
	chatID := m.ChatID()
	photo := StartPhotos[rand.Intn(len(StartPhotos))]
	addServedUser(uid)
	addServedChat(chatID)
	if m.IsPrivate() {
		caption := richImg(photo) + startCaption(uid, name)
		_, _ = sendHTML(Bot, chatID, caption, startHomeKB())
		addBroadcastChat(chatID, "private")
		return nil
	}
	chatTitle := "this chat"
	if m.Chat != nil {
		chatTitle = m.Chat.Title
	}
	caption := richImg(photo) +
		fmt.Sprintf("hey <a href='tg://user?id=%d'>%s</a>\n\nthis is <b>%s</b>\nthanks for adding me in %s.", uid, richEsc(name), richEsc(BotName), richEsc(chatTitle))
	_, _ = sendHTML(Bot, chatID, caption, mixedKeyboard([][][2]string{
		{{"help", "show_help"}, {"about", "about_menu"}},
	}))
	addBroadcastChat(chatID, "group")
	return nil
}

func handleHelp(m *telegram.NewMessage) error {
	if blocked(m) {
		return nil
	}
	_, _ = m.Delete()
	uid := userIDOf(m)
	name := userNameOf(m)
	photo := StartPhotos[rand.Intn(len(StartPhotos))]
	caption := richHeading("help menu", 3) + richImg(photo) +
		richNote(fmt.Sprintf("hey <a href=\"tg://user?id=%d\">%s</a>, tap a category.", uid, richEsc(name)))
	_, _ = sendHTML(Bot, m.ChatID(), caption, helpKB())
	return nil
}

func helpKB() telegram.ReplyMarkup {
	return mixedKeyboard([][][2]string{
		{{"admin", "help_admin"}, {"autoplay", "help_autoplay"}, {"gcast", "help_gcast"}},
		{{"bl-chat", "help_blchat"}, {"bl-users", "help_blusers"}, {"ping", "help_ping"}},
		{{"play", "help_play"}, {"speed", "help_speed"}, {"info", "help_info"}},
		{{"close", "close_help"}},
	})
}

func helpHomeKB() telegram.ReplyMarkup {
	return mixedKeyboard([][][2]string{
		{{"admin", "help_admin"}, {"autoplay", "help_autoplay"}, {"gcast", "help_gcast"}},
		{{"bl-chat", "help_blchat"}, {"bl-users", "help_blusers"}, {"ping", "help_ping"}},
		{{"play", "help_play"}, {"speed", "help_speed"}, {"info", "help_info"}},
		{{"home", "go_back"}},
	})
}

func backKB() telegram.ReplyMarkup {
	return mixedKeyboard([][][2]string{
		{{"back", "show_help"}},
		{{"close", "close_help"}},
	})
}
