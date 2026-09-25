package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

var blockedWords = []string{"blocked"}
var lastCmd = map[int64]time.Time{}
var acceptUpdates atomic.Bool

func enableUpdates() {
	acceptUpdates.Store(true)
}

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
	Bot.On("message:/nothumb", live(handleNoThumb))
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

func messageDate(m *telegram.NewMessage) int64 {
	if m == nil || m.Message == nil {
		return 0
	}
	return int64(m.Message.Date)
}

func isStaleMessage(m *telegram.NewMessage) bool {
	if !acceptUpdates.Load() {
		return true
	}
	if m == nil {
		return true
	}
	ts := messageDate(m)
	text := strings.ToLower(strings.TrimSpace(m.Text()))
	danger := strings.HasPrefix(text, "/start") || strings.HasPrefix(text, "/help") || strings.HasPrefix(text, "/broadcast") || strings.HasPrefix(text, "/gcast")
	if ts == 0 {
		return danger
	}
	return time.Unix(ts, 0).Before(botStartTime.Add(-2 * time.Second))
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

func senderFullName(m *telegram.NewMessage) string {
	if m == nil || m.Sender == nil {
		return "User"
	}
	name := strings.TrimSpace(strings.TrimSpace(m.Sender.FirstName) + " " + strings.TrimSpace(m.Sender.LastName))
	return sanitizeDisplayName(name)
}

func senderUsername(m *telegram.NewMessage) string {
	if m == nil || m.Sender == nil || strings.TrimSpace(m.Sender.Username) == "" {
		return "-"
	}
	return "@" + strings.TrimSpace(m.Sender.Username)
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

func startInner(uid int64, name string) string { return startPrivateHTML(uid, name, BotName) }
func aboutInner() string                       { return startInner(OwnerID, BotName) }
func helpInner(uid int64, name string) string  { return helpMainHTML() }
func startCaption(uid int64, name string) string {
	return startInner(uid, name)
}
func aboutCaption() string { return aboutInner() }
func helpListCaption(uid int64, name string) string {
	return helpInner(uid, name)
}

func sendLogger(text string, markup telegram.ReplyMarkup) {
	if LoggerID == 0 || Bot == nil {
		log.Println("logger skip: LOGGER_ID/LOG_GROUP_ID is 0")
		return
	}
	_, err := Bot.SendMessage(LoggerID, text, &telegram.SendOptions{
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	})
	if err != nil {
		log.Println("logger send failed:", err)
	} else {
		log.Println("logger sent to", LoggerID)
	}
}

func chatTitleOf(chatID int64) string {
	if Bot == nil {
		return "Group"
	}
	raw, err := Bot.GetChat(chatID)
	if err != nil || raw == nil {
		return "Group"
	}
	switch c := raw.(type) {
	case *telegram.Channel:
		if strings.TrimSpace(c.Title) != "" {
			return c.Title
		}
	case *telegram.ChatObj:
		if strings.TrimSpace(c.Title) != "" {
			return c.Title
		}
	}
	return "Group"
}

func playSourceOf(song Song) string {
	return "Searched on Youtube"
}

func playModeOf(song Song, queued bool) string {
	if queued {
		return smallcaps("queue")
	}
	if song.Video {
		return smallcaps("vplay")
	}
	return smallcaps("play")
}

func logPlayAction(chatID int64, song Song, queued bool) {
	if LoggerID == 0 || Bot == nil || chatID == 0 {
		log.Println("play log skip: logger id empty")
		return
	}
	name := song.Requester
	if name == "" {
		name = "User"
	}
	title := song.Title
	if title == "" {
		title = "Unknown"
	}
	head := smallcaps("new play log")
	if queued {
		head = smallcaps("new queue log")
	}
	body := "<blockquote expandable><b>" + head + "</b>\n\n" +
		smallcaps("user") + " : " + richEsc(name) + " [<code>" + fmt.Sprintf("%d", song.RequesterID) + "</code>]\n" +
		smallcaps("group") + " : " + richEsc(chatTitleOf(chatID)) + "\n" +
		smallcaps("group id") + " : <code>" + fmt.Sprintf("%d", chatID) + "</code>\n" +
		smallcaps("query/song") + " : " + richEsc(title) + "\n" +
		smallcaps("source") + " : " + richEsc(playSourceOf(song)) + "\n" +
		smallcaps("mode") + " : " + playModeOf(song, queued) + "</blockquote>"
	var kb telegram.ReplyMarkup
	if song.RequesterID != 0 {
		kb = mixedKeyboard([][][2]string{{
			{"OPEN PROFILE", fmt.Sprintf("tg://user?id=%d", song.RequesterID)},
		}})
	}
	sendLogger(body, kb)
}

func logNewUserStart(m *telegram.NewMessage) {
	if LoggerID == 0 || Bot == nil || m == nil {
		log.Println("start log skip: logger id empty or no message")
		return
	}
	if !m.IsPrivate() {
		return
	}
	uid := userIDOf(m)
	if uid == 0 {
		return
	}
	name := senderFullName(m)
	uname := senderUsername(m)
	total := getServedUsersCount()
	body := "<blockquote expandable><b>NEW USER STARTED BOT</b>\n\n" +
		"<b>NAME</b> — <code>" + richEsc(name) + "</code>\n" +
		"<b>U NAME</b> — <code>" + richEsc(uname) + "</code>\n" +
		"<b>U ID</b> — <code>" + fmt.Sprintf("%d", uid) + "</code>\n" +
		"<b>TOTAL USER</b> — <code>" + fmt.Sprintf("%d", total) + "</code></blockquote>"
	kb := mixedKeyboard([][][2]string{{
		{"OPEN PROFILE", fmt.Sprintf("tg://user?id=%d", uid)},
	}})
	sendLogger(body, kb)
}

func handleStart(m *telegram.NewMessage) error {
	if blocked(m) {
		return nil
	}
	_, _ = m.Delete()
	uid := userIDOf(m)
	name := userNameOf(m)
	chatID := m.ChatID()
	_ = addServedUser(uid)
	addServedChat(chatID)
	arg := strings.ToLower(cmdArgs(m))
	if m.IsPrivate() {
		go logNewUserStart(m)
		if arg == "pm_help" {
			_, _ = sendQuotedPhoto(chatID, helpInner(uid, name), GetHelpMarkup())
		} else {
			_, _ = sendQuotedPhoto(chatID, startInner(uid, name), GetStartMarkup())
		}
		addBroadcastChat(chatID, "private")
		return nil
	}
	_, _ = sendQuotedPhoto(chatID, startGroupHTML(), GetGroupStartMarkup())
	addBroadcastChat(chatID, "group")
	return nil
}

func handleHelp(m *telegram.NewMessage) error {
	if blocked(m) {
		return nil
	}
	_, _ = m.Delete()
	if !m.IsPrivate() {
		_, _ = sendQuotedPhoto(m.ChatID(), helpPrivateOnlyHTML(), GetGroupStartMarkup())
		return nil
	}
	_, _ = sendQuotedPhoto(m.ChatID(), helpInner(userIDOf(m), userNameOf(m)), GetHelpMarkup())
	return nil
}
