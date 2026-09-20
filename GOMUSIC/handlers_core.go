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
		{{"➕ " + smallcaps("add me in your group") + " ➕", BotLink + "?startgroup=true"}},
		{{smallcaps("owner"), fmt.Sprintf("tg://user?id=%d", OwnerID)}, {smallcaps("about"), "about_menu"}},
		{{smallcaps("support"), SupportGroup}, {smallcaps("update"), UpdatesChannel}},
		{{smallcaps("help and commands"), "show_help"}},
	})
}

func aboutKB() telegram.ReplyMarkup {
	return mixedKeyboard([][][2]string{
		{{smallcaps("back"), "go_back"}},
	})
}

func startCaption(uid int64, name string) string {
	mention := fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", uid, richEsc(name))
	body := smallcaps("hey") + " " + mention + "\n\n" +
		smallcaps("i am a high quality fast music bot.") + "\n" +
		smallcaps("add me to your group and enjoy audio / video streaming.") + "\n\n" +
		smallcaps("use the buttons below.")
	return wrapBQ(body)
}

func aboutCaption() string {
	body := smallcaps("about") + "\n\n" +
		smallcaps("high quality telegram music bot.") + "\n" +
		smallcaps("supports audio and video streaming.") + "\n" +
		smallcaps("powered by go + gogram + ntgcalls.") + "\n\n" +
		smallcaps("bot") + " : <code>" + richEsc(BotName) + "</code>\n" +
		smallcaps("version") + " : <code>GOMUSIC v2</code>\n" +
		smallcaps("language") + " : <code>Go</code>\n" +
		smallcaps("telegram") + " : <code>gogram v1.7.10</code>\n" +
		smallcaps("calls") + " : <code>ntgcalls v2.2.5</code>\n" +
		smallcaps("player") + " : <code>ffmpeg + yt-dlp</code>\n" +
		smallcaps("runtime") + " : <code>" + runtime.Version() + "</code>\n\n" +
		smallcaps("add me in your group and start playing.")
	return wrapBQ(body)
}

func helpListCaption(uid int64, name string) string {
	mention := fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", uid, richEsc(name))
	body := smallcaps("help menu") + "\n\n" +
		smallcaps("hey") + " " + mention + "\n" +
		smallcaps("tap any command button below to see how to use it.")
	return wrapBQ(body)
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
	caption := richImg(photo) + wrapBQ(
		smallcaps("hey")+" "+fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", uid, richEsc(name))+"\n\n"+
			smallcaps("this is")+" <b>"+richEsc(BotName)+"</b>\n"+
			smallcaps("thanks for adding me in")+" "+richEsc(chatTitle)+".",
	)
	_, _ = sendHTML(Bot, chatID, caption, mixedKeyboard([][][2]string{
		{{smallcaps("help"), "show_help"}, {smallcaps("about"), "about_menu"}},
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
	caption := richImg(photo) + helpListCaption(uid, name)
	_, _ = sendHTML(Bot, m.ChatID(), caption, helpKB())
	return nil
}

func helpKB() telegram.ReplyMarkup {
	return mixedKeyboard([][][2]string{
		{{smallcaps("admin"), "help_admin"}, {smallcaps("autoplay"), "help_autoplay"}, {smallcaps("gcast"), "help_gcast"}},
		{{smallcaps("bl-chat"), "help_blchat"}, {smallcaps("bl-users"), "help_blusers"}, {smallcaps("ping"), "help_ping"}},
		{{smallcaps("play"), "help_play"}, {smallcaps("speed"), "help_speed"}, {smallcaps("info"), "help_info"}},
		{{smallcaps("close"), "close_help"}},
	})
}

func helpHomeKB() telegram.ReplyMarkup {
	return mixedKeyboard([][][2]string{
		{{smallcaps("admin"), "help_admin"}, {smallcaps("autoplay"), "help_autoplay"}, {smallcaps("gcast"), "help_gcast"}},
		{{smallcaps("bl-chat"), "help_blchat"}, {smallcaps("bl-users"), "help_blusers"}, {smallcaps("ping"), "help_ping"}},
		{{smallcaps("play"), "help_play"}, {smallcaps("speed"), "help_speed"}, {smallcaps("info"), "help_info"}},
		{{smallcaps("back"), "go_back"}},
	})
}

func backKB() telegram.ReplyMarkup {
	return mixedKeyboard([][][2]string{
		{{smallcaps("back"), "show_help"}},
		{{smallcaps("close"), "close_help"}},
	})
}
