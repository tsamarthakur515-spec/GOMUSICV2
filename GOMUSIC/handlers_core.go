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

func live(h func(*telegram.NewMessage) error) func(*telegram.NewMessage) error {
	return func(m *telegram.NewMessage) error {
		if isStaleMessage(m) {
			return nil
		}
		return h(m)
	}
}

func registerHandlers() {
	Bot.On("message:/start", live(handleStart))
	Bot.On("message:/help", live(handleHelp))
	Bot.On("message:/play", live(handlePlay))
	Bot.On("message:/vplay", live(handleVPlay))
	Bot.On("message:/pause", live(handlePause))
	Bot.On("message:/resume", live(handleResume))
	Bot.On("message:/skip", live(handleSkip))
	Bot.On("message:/stop", live(handleStop))
	Bot.On("message:/end", live(handleStop))
	Bot.On("message:/clear", live(handleClear))
	Bot.On("message:/queue", live(handleQueue))
	Bot.On("message:/reboot", live(handleReboot))
	Bot.On("message:/ping", live(handlePing))
	Bot.On("message:/id", live(handleID))
	Bot.On("message:/autoplay", live(handleAutoplay))
	Bot.On("message:/speed", live(handleSpeed))
	Bot.On("message:/speedreset", live(handleSpeedReset))
	Bot.On("message:/bass", live(handleBass))
	Bot.On("message:/bassoff", live(handleBassOff))
	Bot.On("message:/effecton", live(handleEffectOn))
	Bot.On("message:/effectoff", live(handleEffectOff))
	Bot.On("message:/effects", live(handleEffects))
	Bot.On("message:/seek", live(handleSeek))
	Bot.On("message:/seekback", live(handleSeekBack))
	Bot.On("message:/gblock", live(handleGBlock))
	Bot.On("message:/gunblock", live(handleGUnblock))
	Bot.On("message:/ublock", live(handleUBlock))
	Bot.On("message:/uunblock", live(handleUUnblock))
	Bot.On("message:/blocklist", live(handleBlocklist))
	Bot.On("message:/broadcast", live(handleBroadcast))
	Bot.On("message:/gcast", live(handleBroadcast))
	Bot.On("message:/stats", live(handleStats))
	Bot.On(telegram.OnCallbackQuery, handleCallbackQuery)
}

func isStaleMessage(m *telegram.NewMessage) bool {
	if m == nil {
		return true
	}
	var ts int64
	if m.Message != nil {
		if obj, ok := m.Message.(*telegram.MessageObj); ok && obj.Date > 0 {
			ts = int64(obj.Date)
		}
	}
	if ts == 0 {
		return false
	}
	return time.Unix(ts, 0).Before(botStartTime.Add(-3 * time.Second))
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
	if isStaleMessage(m) {
		return true
	}
	if isGroupBlocked(m.ChatID()) {
		return true
	}
	return isUserBlockedDB(userIDOf(m))
}

func pickStartPhoto() string {
	if len(StartPhotos) == 0 {
		return ""
	}
	return StartPhotos[rand.Intn(len(StartPhotos))]
}

func startInner(uid int64, name string) string {
	mention := fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", uid, richEsc(name))
	return "<blockquote><b>" + smallcaps("hey") + "</b> " + mention + ",</blockquote>\n" +
		"<blockquote expandable><b>" + smallcaps("i am a high quality fast music bot.") + "</b>\n" +
		"<b>" + smallcaps("add me to your group and enjoy audio / video streaming.") + "</b>\n" +
		"<b>" + smallcaps("use the buttons below.") + "</b></blockquote>"
}

func aboutInner() string {
	return "<blockquote><b>" + smallcaps("about") + "</b></blockquote>\n" +
		"<blockquote expandable><b>" + smallcaps("high quality telegram music bot.") + "</b>\n" +
		"<b>" + smallcaps("supports audio and video streaming.") + "</b>\n" +
		"<b>" + smallcaps("powered by go + gogram + ntgcalls.") + "</b>\n\n" +
		"<b>" + smallcaps("bot") + "</b> : <code>" + richEsc(BotName) + "</code>\n" +
		"<b>" + smallcaps("version") + "</b> : <code>GOMUSIC v2</code>\n" +
		"<b>" + smallcaps("language") + "</b> : <code>Go</code>\n" +
		"<b>" + smallcaps("telegram") + "</b> : <code>gogram v1.7.10</code>\n" +
		"<b>" + smallcaps("calls") + "</b> : <code>ntgcalls v2.2.5</code>\n" +
		"<b>" + smallcaps("player") + "</b> : <code>ffmpeg + yt-dlp</code>\n" +
		"<b>" + smallcaps("runtime") + "</b> : <code>" + runtime.Version() + "</code></blockquote>"
}

func helpInner(uid int64, name string) string {
	mention := fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", uid, richEsc(name))
	return "<blockquote><b>" + smallcaps("help menu") + "</b></blockquote>\n\n" +
		"<blockquote><b>" + smallcaps("hey") + "</b> " + mention + "</blockquote>\n" +
		"<blockquote><b>" + smallcaps("tap any command button below to see how to use it.") + "</b></blockquote>"
}

func startCaption(uid int64, name string) string { return startInner(uid, name) }
func aboutCaption() string                       { return aboutInner() }
func helpListCaption(uid int64, name string) string {
	return helpInner(uid, name)
}

func handleStart(m *telegram.NewMessage) error {
	if blocked(m) {
		return nil
	}
	_, _ = m.Delete()
	uid := userIDOf(m)
	name := userNameOf(m)
	chatID := m.ChatID()
	addServedUser(uid)
	addServedChat(chatID)
	if m.IsPrivate() {
		_, _ = sendQuotedPhoto(chatID, startInner(uid, name), GetStartMarkup())
		addBroadcastChat(chatID, "private")
		return nil
	}
	chatTitle := "this chat"
	if m.Chat != nil {
		chatTitle = m.Chat.Title
	}
	inner := "<blockquote><b>" + smallcaps("hey") + "</b> " + fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", uid, richEsc(name)) + "</blockquote>\n" +
		"<blockquote expandable><b>" + smallcaps("this is") + " " + richEsc(BotName) + "</b>\n" +
		"<b>" + smallcaps("thanks for adding me in") + " " + richEsc(chatTitle) + ".</b></blockquote>"
	_, _ = sendQuotedPhoto(chatID, inner, GetGroupStartMarkup())
	addBroadcastChat(chatID, "group")
	return nil
}

func handleHelp(m *telegram.NewMessage) error {
	if blocked(m) {
		return nil
	}
	_, _ = m.Delete()
	_, _ = sendQuotedPhoto(m.ChatID(), helpInner(userIDOf(m), userNameOf(m)), GetHelpMarkup())
	return nil
}
