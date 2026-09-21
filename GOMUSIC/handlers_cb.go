package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

var (
	cbOnce   sync.Map
	cbOnceMu sync.Mutex
)

func handleCallback(m *telegram.NewMessage) error { return nil }

func alreadyHandledCallback(cb *telegram.CallbackQuery) bool {
	if cb == nil {
		return true
	}
	key := fmt.Sprintf("%d:%d:%d:%s", cb.QueryID, cb.ChatID, cb.MessageID, string(cb.Data))
	cbOnceMu.Lock()
	defer cbOnceMu.Unlock()
	if t, ok := cbOnce.Load(key); ok {
		if time.Since(t.(time.Time)) < 2*time.Second {
			return true
		}
	}
	cbOnce.Store(key, time.Now())
	return false
}

func handleCallbackQuery(cb *telegram.CallbackQuery) error {
	if cb == nil || cb.Sender == nil {
		return nil
	}
	if alreadyHandledCallback(cb) {
		return nil
	}
	if isUserBlockedDB(cb.Sender.ID) {
		_, _ = cb.Answer("")
		return nil
	}
	data := string(cb.Data)
	chatID := cb.ChatID
	msg, _ := cb.GetMessage()
	alert := &telegram.CallbackOptions{Alert: true}
	switch data {
	case "pause":
		if _, err := Calls.Pause(chatID); err != nil {
			_, _ = cb.Answer("failed to pause", alert)
			return nil
		}
		_, _ = cb.Answer("paused")
	case "resume":
		if _, err := Calls.Resume(chatID); err != nil {
			_, _ = cb.Answer("failed to resume", alert)
			return nil
		}
		_, _ = cb.Answer("resumed")
	case "skip":
		if queueSize(chatID) == 0 {
			_, _ = cb.Answer("queue is empty", alert)
			return nil
		}
		skipped := popCurrent(chatID)
		_ = Calls.Stop(chatID)
		time.Sleep(2 * time.Second)
		if skipped != nil {
			deleteFile(skipped.FilePath)
		}
		if nxt := peekCurrent(chatID); nxt != nil {
			_, _ = cb.Answer("skipped")
			_ = playSong(chatID, nil, *nxt)
		} else {
			_, _ = cb.Answer("queue empty", alert)
		}
	case "stop":
		leaveVC(chatID)
		_, _ = cb.Answer("stopped")
		if msg != nil {
			_, _ = msg.Delete()
		}
	case "close_panel":
		_, _ = cb.Answer("")
		if msg != nil {
			_, _ = msg.Delete()
		}
	case "progress":
		cur := peekCurrent(chatID)
		title := "nothing playing"
		if cur != nil {
			title = shortTitle(cur.Title, 28)
		}
		_, _ = cb.Answer(title, alert)
	case "seek_back", "seek_fwd":
		delta := 10 * time.Second
		if data == "seek_back" {
			delta = -10 * time.Second
		}
		if err := Calls.SeekBy(chatID, delta.Milliseconds()); err != nil {
			_, _ = cb.Answer("seek failed", alert)
			return nil
		}
		_, _ = cb.Answer("seeked")
	case "clear":
		clearQueue(chatID)
		_, _ = cb.Answer("queue cleared")
	case "noop":
		_, _ = cb.Answer("")
	case "close_help":
		_, _ = cb.Answer("")
		if msg != nil {
			_, _ = msg.Delete()
		}
	case "about_menu":
		_, _ = cb.Answer("")
		_ = editMenu(cb, aboutCaption(), aboutKB())
	case "go_back":
		_, _ = cb.Answer("")
		uid := cb.Sender.ID
		name := sanitizeDisplayName(cb.Sender.FirstName)
		_ = editMenu(cb, startCaption(uid, name), startHomeKB())
	case "show_help":
		_, _ = cb.Answer("")
		uid := cb.Sender.ID
		name := sanitizeDisplayName(cb.Sender.FirstName)
		_ = editMenu(cb, helpListCaption(uid, name), helpHomeKB())
	default:
		if strings.HasPrefix(data, "help_") {
			_, _ = cb.Answer("")
			if h, ok := helpTexts[data]; ok {
				body := smallcaps(h.title) + "\n\n" + smallcaps(h.desc) + "\n\n" + richTable([]string{"command", "description"}, h.rows)
				_ = editMenu(cb, wrapBQ(body), backKB())
			}
		}
	}
	return nil
}

type helpCat struct {
	title, desc string
	rows        [][]string
}

var helpTexts = map[string]helpCat{
	"help_admin":    {title: "admin commands", desc: "core playback controls for chat admins.", rows: [][]string{{"/pause", "pause"}, {"/resume", "resume"}, {"/skip", "skip"}, {"/stop", "stop"}, {"/clear", "clear queue"}, {"/seek", "seek forward"}, {"/seekback", "seek back"}, {"/reboot", "reset chat"}}},
	"help_autoplay": {title: "autoplay commands", desc: "keep the queue going automatically.", rows: [][]string{{"/autoplay query", "start autoplay"}, {"/end", "stop autoplay"}}},
	"help_gcast":    {title: "gcast commands", desc: "broadcast to every served chat (owner only).", rows: [][]string{{"/broadcast", "send text to all chats"}}},
	"help_blchat":   {title: "block chat commands", desc: "block or unblock groups (owner only).", rows: [][]string{{"/gblock", "block group"}, {"/gunblock", "unblock group"}}},
	"help_blusers":  {title: "block user commands", desc: "block or unblock users (owner only).", rows: [][]string{{"/ublock", "block user"}, {"/uunblock", "unblock user"}}},
	"help_ping":     {title: "ping commands", desc: "latency and system diagnostics.", rows: [][]string{{"/ping", "bot latency"}, {"/stats", "full stats"}}},
	"help_play":     {title: "play commands", desc: "start audio or video playback in a voice chat.", rows: [][]string{{"/play", "play audio"}, {"/vplay", "play video"}}},
	"help_speed":    {title: "speed and effects", desc: "adjust playback speed and audio effects.", rows: [][]string{{"/speed", "change speed"}, {"/bass", "bass boost"}, {"/effects", "status"}}},
	"help_info":     {title: "info commands", desc: "bot, chat, and user information.", rows: [][]string{{"/id", "get ids"}, {"/stats", "stats"}}},
}

func notifyOwner(me *telegram.UserObj, asst string) {
	if LoggerID == 0 || me == nil {
		return
	}
	content := wrapBQ(smallcaps("bot started") + "\n\n" +
		smallcaps("bot") + " : @" + richEsc(me.Username) + "\n" +
		smallcaps("assistant") + " : @" + richEsc(asst))
	if _, err := sendHTML(Bot, LoggerID, content, nil); err != nil {
		log.Println("Logger Notification Error :", err)
	}
}
