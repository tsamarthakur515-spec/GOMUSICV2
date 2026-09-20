package main

import (
	"fmt"
	"math/rand"
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
		{{"Add me", BotLink + "?startgroup=true"}},
		{{"Support", SupportGroup}, {"Updates", UpdatesChannel}},
		{{"Help & Commands", "show_help"}},
		{{"Owner", fmt.Sprintf("tg://user?id=%d", OwnerID)}},
	})
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
		caption := richImg(photo) +
			richNote(fmt.Sprintf("<p>hey <a href='tg://user?id=%d'>%s</a>, welcome aboard!</p><p>I am <b>%s</b> — a Telegram music player bot.</p>", uid, richEsc(name), richEsc(BotName))) +
			richDetails("key features", richTable(nil, [][]string{
				{"streaming", "play audio in voice chats"},
				{"autoplay", "keeps the queue going automatically"},
			}), true) +
			richNote("powered by Shizu Music")
		_, _ = sendHTML(Bot, chatID, caption, startHomeKB())
		addBroadcastChat(chatID, "private")
		return nil
	}
	chatTitle := "this chat"
	if m.Chat != nil {
		chatTitle = m.Chat.Title
	}
	caption := richImg(photo) +
		fmt.Sprintf("<p>hey <a href='tg://user?id=%d'>%s</a>, this is <b>%s</b></p>", uid, richEsc(name), richEsc(BotName)) +
		richNote(fmt.Sprintf("thanks for adding me in %s.", richEsc(chatTitle)))
	_, _ = sendHTML(Bot, chatID, caption, mixedKeyboard([][][2]string{
		{{"help", "show_help"}},
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
	caption := richHeading("choose a category", 3) + richImg(photo) +
		richNote(fmt.Sprintf(`<p>hey <a href="tg://user?id=%d">%s</a>, pick a category below.</p>`, uid, richEsc(name)))
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
