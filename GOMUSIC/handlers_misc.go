package main

import (
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

func handlePing(m *telegram.NewMessage) error {
	if blocked(m) {
		return nil
	}
	start := time.Now()
	pm, _ := sendHTML(Bot, m.ChatID(), richHeading(richEsc(BotName)+" is pinging...", 3), nil)
	latency := time.Since(start).Milliseconds()
	uptime := time.Since(botStartTime).Truncate(time.Second).String()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	if pm != nil {
		_, _ = pm.Delete()
	}
	_, _ = sendHTML(Bot, m.ChatID(), richHeading(fmt.Sprintf("pong : %dms", latency), 3)+richImg(PingImgURL)+richKVTable([][2]string{
		{"uptime", "<code>" + uptime + "</code>"},
		{"ram", fmt.Sprintf("<code>%.2f MB</code>", float64(ms.Alloc)/1024/1024)},
		{"cpu", "<code>" + runtime.GOARCH + "</code>"},
	}), mixedKeyboard([][][2]string{{{ "support", SupportGroup }}}))
	return nil
}

func handleRepo(m *telegram.NewMessage) error {
	return nil
}

func handleID(m *telegram.NewMessage) error {
	if blocked(m) {
		return nil
	}
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("id info", 3)+richKVTable([][2]string{
		{"user id", fmt.Sprintf("<code>%d</code>", userIDOf(m))},
		{"chat id", fmt.Sprintf("<code>%d</code>", m.ChatID())},
		{"msg id", fmt.Sprintf("<code>%d</code>", m.ID)},
	}), nil)
	return nil
}

func handleAutoplay(m *telegram.NewMessage) error {
	if blocked(m) || m.IsPrivate() {
		return nil
	}
	if !isAuthorized(m) {
		_, _ = sendHTML(Bot, m.ChatID(), richHeading("admin only", 3), nil)
		return nil
	}
	query := cmdArgs(m)
	chatID := m.ChatID()
	if query == "" {
		_, _ = sendHTML(Bot, chatID, richHeading("autoplay usage", 3)+richNote("<code>/autoplay artist name</code>"), nil)
		return nil
	}
	pm, _ := sendHTML(Bot, chatID, richHeading("setting up autoplay...", 3), nil)
	n := startAutoplay(chatID, query, userNameOf(m), userIDOf(m))
	_ = editHTML(pm, richHeading("autoplay started", 3)+richKVTable([][2]string{
		{"query", "<code>" + richEsc(query) + "</code>"},
		{"queued", fmt.Sprintf("<code>%d</code>", n)},
	}), nil)
	if peekCurrent(chatID) != nil && n > 0 {
		return playSong(chatID, pm, *peekCurrent(chatID))
	}
	return nil
}

func handleSpeed(m *telegram.NewMessage) error {
	if blocked(m) || !isAuthorized(m) {
		return nil
	}
	v, err := parseFloatArg(cmdArgs(m))
	if err != nil || v < 0.25 || v > 4 {
		_, _ = sendHTML(Bot, m.ChatID(), richHeading("usage", 3)+richNote("<code>/speed 1.5</code>"), nil)
		return nil
	}
	setSpeed(m.ChatID(), v)
	setEffectsEnabled(m.ChatID(), true)
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("speed updated", 3)+richKVTable([][2]string{{"speed", fmt.Sprintf("<code>%.2fx</code>", v)}}), nil)
	return nil
}

func handleSpeedReset(m *telegram.NewMessage) error {
	if blocked(m) || !isAuthorized(m) {
		return nil
	}
	setSpeed(m.ChatID(), 1)
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("speed reset", 3)+richNote("back to 1.0x"), nil)
	return nil
}

func handleBass(m *telegram.NewMessage) error {
	if blocked(m) || !isAuthorized(m) {
		return nil
	}
	n, err := strconv.Atoi(cmdArgs(m))
	if err != nil || n < 1 || n > 20 {
		_, _ = sendHTML(Bot, m.ChatID(), richHeading("usage", 3)+richNote("<code>/bass 10</code>"), nil)
		return nil
	}
	setBass(m.ChatID(), n)
	setEffectsEnabled(m.ChatID(), true)
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("bass updated", 3)+richKVTable([][2]string{{"bass", fmt.Sprintf("<code>%d dB</code>", n)}}), nil)
	return nil
}

func handleBassOff(m *telegram.NewMessage) error {
	if blocked(m) || !isAuthorized(m) {
		return nil
	}
	setBass(m.ChatID(), 0)
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("bass off", 3), nil)
	return nil
}

func handleEffectOn(m *telegram.NewMessage) error {
	if blocked(m) || !isAuthorized(m) {
		return nil
	}
	setEffectsEnabled(m.ChatID(), true)
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("effects on", 3), nil)
	return nil
}

func handleEffectOff(m *telegram.NewMessage) error {
	if blocked(m) || !isAuthorized(m) {
		return nil
	}
	setEffectsEnabled(m.ChatID(), false)
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("effects off", 3), nil)
	return nil
}

func handleEffects(m *telegram.NewMessage) error {
	if blocked(m) {
		return nil
	}
	s := getEffects(m.ChatID())
	on := "off"
	if s.Enabled {
		on = "on"
	}
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("effects", 3)+richKVTable([][2]string{
		{"status", "<code>" + on + "</code>"},
		{"speed", fmt.Sprintf("<code>%.2fx</code>", s.Speed)},
		{"bass", fmt.Sprintf("<code>%d dB</code>", s.Bass)},
	}), nil)
	return nil
}

func handleSeek(m *telegram.NewMessage) error {
	if blocked(m) || !isAuthorized(m) {
		return nil
	}
	n, _ := strconv.Atoi(cmdArgs(m))
	if n <= 0 {
		_, _ = sendHTML(Bot, m.ChatID(), richHeading("usage", 3)+richNote("<code>/seek 15</code>"), nil)
		return nil
	}
	setSeekState(m.ChatID(), getSeekState(m.ChatID())+n)
	_ = Calls.SeekBy(m.ChatID(), int64(n)*1000)
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("seek", 3)+richKVTable([][2]string{{"sec", fmt.Sprintf("<code>+%d</code>", n)}}), nil)
	return nil
}

func handleSeekBack(m *telegram.NewMessage) error {
	if blocked(m) || !isAuthorized(m) {
		return nil
	}
	n, _ := strconv.Atoi(cmdArgs(m))
	if n <= 0 {
		_, _ = sendHTML(Bot, m.ChatID(), richHeading("usage", 3)+richNote("<code>/seekback 15</code>"), nil)
		return nil
	}
	_ = Calls.SeekBy(m.ChatID(), -int64(n)*1000)
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("seek back", 3)+richKVTable([][2]string{{"sec", fmt.Sprintf("<code>-%d</code>", n)}}), nil)
	return nil
}

func handleGBlock(m *telegram.NewMessage) error {
	if !isOwner(userIDOf(m)) {
		return nil
	}
	id := m.ChatID()
	if a := cmdArgs(m); a != "" {
		v, err := strconv.ParseInt(a, 10, 64)
		if err != nil {
			return nil
		}
		id = v
	}
	blockGroup(id)
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("group blocked", 3)+richKVTable([][2]string{{"id", fmt.Sprintf("<code>%d</code>", id)}}), nil)
	return nil
}

func handleGUnblock(m *telegram.NewMessage) error {
	if !isOwner(userIDOf(m)) {
		return nil
	}
	id := m.ChatID()
	if a := cmdArgs(m); a != "" {
		v, err := strconv.ParseInt(a, 10, 64)
		if err == nil {
			id = v
		}
	}
	unblockGroup(id)
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("group unblocked", 3)+richKVTable([][2]string{{"id", fmt.Sprintf("<code>%d</code>", id)}}), nil)
	return nil
}

func handleUBlock(m *telegram.NewMessage) error {
	if !isOwner(userIDOf(m)) {
		return nil
	}
	id, _ := strconv.ParseInt(cmdArgs(m), 10, 64)
	if id == 0 {
		_, _ = sendHTML(Bot, m.ChatID(), richHeading("usage", 3)+richNote("<code>/ublock user_id</code>"), nil)
		return nil
	}
	blockUser(id)
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("user blocked", 3)+richKVTable([][2]string{{"id", fmt.Sprintf("<code>%d</code>", id)}}), nil)
	return nil
}

func handleUUnblock(m *telegram.NewMessage) error {
	if !isOwner(userIDOf(m)) {
		return nil
	}
	id, _ := strconv.ParseInt(cmdArgs(m), 10, 64)
	if id == 0 {
		return nil
	}
	unblockUser(id)
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("user unblocked", 3)+richKVTable([][2]string{{"id", fmt.Sprintf("<code>%d</code>", id)}}), nil)
	return nil
}

func handleBlocklist(m *telegram.NewMessage) error {
	if !isOwner(userIDOf(m)) {
		return nil
	}
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("blocklist", 3)+richKVTable([][2]string{
		{"groups", fmt.Sprintf("<code>%d</code>", len(getBlockedGroups()))},
		{"users", fmt.Sprintf("<code>%d</code>", len(getBlockedUsers()))},
	}), nil)
	return nil
}

func handleBroadcast(m *telegram.NewMessage) error {
	if !isOwner(userIDOf(m)) {
		return nil
	}
	text := cmdArgs(m)
	if text == "" {
		_, _ = sendHTML(Bot, m.ChatID(), richHeading("usage", 3)+richNote("<code>/broadcast text</code>"), nil)
		return nil
	}
	n := 0
	for _, id := range getBroadcastChats() {
		if _, err := sendHTML(Bot, id, text, nil); err == nil {
			n++
		}
	}
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("broadcast done", 3)+richKVTable([][2]string{{"sent", fmt.Sprintf("<code>%d</code>", n)}}), nil)
	return nil
}

func handleStats(m *telegram.NewMessage) error {
	if !isOwner(userIDOf(m)) {
		return nil
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("stats", 3)+richKVTable([][2]string{
		{"chats", fmt.Sprintf("<code>%d</code>", getServedChatsCount())},
		{"users", fmt.Sprintf("<code>%d</code>", getServedUsersCount())},
		{"plays", fmt.Sprintf("<code>%d</code>", getTotalPlays())},
		{"ram", fmt.Sprintf("<code>%.2f MB</code>", float64(ms.Alloc)/1024/1024)},
		{"go", "<code>" + runtime.Version() + "</code>"},
	}), nil)
	return nil
}
