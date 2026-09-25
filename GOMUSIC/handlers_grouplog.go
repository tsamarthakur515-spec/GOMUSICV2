package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

var memLogOnce sync.Map

func canonChatID(id int64) int64 {
	if id == 0 {
		return 0
	}
	s := strconv.FormatInt(id, 10)
	if strings.HasPrefix(s, "-100") {
		return id
	}
	raw := id
	if raw < 0 {
		raw = -raw
	}
	if raw >= 1000000000 && raw < 1000000000000 {
		return -1000000000000 - raw
	}
	return id
}

func profileURL(uid int64, username string) string {
	u := strings.TrimPrefix(strings.TrimSpace(username), "@")
	if u != "" && u != "-" {
		return "https://t.me/" + u
	}
	return ""
}

func profileMarkup(uid int64, username string) telegram.ReplyMarkup {
	url := profileURL(uid, username)
	if url == "" {
		return nil
	}
	return mixedKeyboard([][][2]string{{
		{"OPEN PROFILE", url},
	}})
}

func mentionHTML(uid int64, name string) string {
	if name == "" {
		name = "User"
	}
	if uid == 0 {
		return richEsc(name)
	}
	return fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, uid, richEsc(name))
}

func userBits(u *telegram.UserObj) (name, uname string, uid int64) {
	name = "User"
	uname = "-"
	if u == nil {
		return
	}
	uid = u.ID
	n := strings.TrimSpace(strings.TrimSpace(u.FirstName) + " " + strings.TrimSpace(u.LastName))
	if n != "" {
		name = sanitizeDisplayName(n)
	}
	if strings.TrimSpace(u.Username) != "" {
		uname = "@" + strings.TrimSpace(u.Username)
	}
	return
}

func groupTypeAndLink(chatID int64) (gtype, link, title string) {
	gtype = smallcaps("private")
	link = "-"
	title = "Group"
	if Bot == nil {
		return
	}
	ids := []int64{chatID, canonChatID(chatID)}
	for _, id := range ids {
		if ch, err := Bot.GetChannel(id); err == nil && ch != nil {
			if strings.TrimSpace(ch.Title) != "" {
				title = ch.Title
			}
			if uname := strings.TrimSpace(ch.Username); uname != "" {
				gtype = smallcaps("public")
				link = "https://t.me/" + uname
			}
			return
		}
	}
	for _, id := range ids {
		if chat, err := Bot.GetChat(id); err == nil && chat != nil && strings.TrimSpace(chat.Title) != "" {
			title = chat.Title
			return
		}
	}
	return
}

func logBotMembership(chatID int64, byID int64, byName, byUser string, kicked bool) {
	chatID = canonChatID(chatID)
	if LoggerID == 0 || chatID == 0 || chatID == LoggerID || chatID == canonChatID(LoggerID) {
		return
	}
	if byName == "" {
		byName = "User"
	}
	if byUser == "" {
		byUser = "-"
	}
	key := fmt.Sprintf("%d:%v", chatID, kicked)
	if v, ok := memLogOnce.Load(key); ok {
		if t, ok := v.(time.Time); ok && time.Since(t) < 15*time.Second {
			return
		}
	}
	memLogOnce.Store(key, time.Now())
	gtype, link, title := groupTypeAndLink(chatID)
	head := smallcaps("bot added in new group")
	if kicked {
		head = smallcaps("bot was kick from group")
	}
	body := "<blockquote expandable><b>" + head + "</b>\n\n" +
		smallcaps("title") + " : " + richEsc(title) + "\n" +
		smallcaps("name") + " : " + mentionHTML(byID, byName) + "\n" +
		smallcaps("u name") + " : <code>" + richEsc(byUser) + "</code>\n" +
		smallcaps("u id") + " : <code>" + fmt.Sprintf("%d", byID) + "</code>\n" +
		smallcaps("g name") + " : " + richEsc(title) + "\n" +
		smallcaps("g id") + " : <code>" + fmt.Sprintf("%d", chatID) + "</code>\n" +
		smallcaps("g type") + " : " + gtype
	if link != "-" {
		body += "\n" + smallcaps("link") + " : <a href=\"" + richEsc(link) + "\">" + richEsc(link) + "</a>"
	} else {
		body += "\n" + smallcaps("link") + " : <code>private</code>"
	}
	body += "</blockquote>"
	log.Println("group membership log kicked=", kicked, "chat=", chatID, "by=", byID, "user=", byUser)
	sendLogger(body, profileMarkup(byID, byUser))
}

func botSelfID() int64 {
	if Bot == nil {
		return 0
	}
	me, err := Bot.GetMe()
	if err != nil || me == nil {
		return 0
	}
	return me.ID
}

func handleParticipant(p *telegram.ParticipantUpdate) error {
	if p == nil || p.User == nil {
		return nil
	}
	self := botSelfID()
	if self == 0 || p.User.ID != self {
		return nil
	}
	added := p.IsAdded() || p.IsJoined()
	kicked := p.IsKicked() || p.IsLeft() || p.IsBanned()
	log.Println("participant self added=", added, "kicked=", kicked, "chat=", p.ChannelID())
	if !added && !kicked {
		return nil
	}
	chatID := canonChatID(p.ChannelID())
	byName, byUser, byID := userBits(p.Actor)
	if byID == 0 {
		byName, byUser, byID = userBits(p.User)
	}
	if added {
		addServedChat(chatID)
		addBroadcastChat(chatID, "group")
	}
	go logBotMembership(chatID, byID, byName, byUser, kicked)
	return nil
}

func handleServiceMessage(m *telegram.NewMessage) error {
	if m == nil || m.IsPrivate() || !m.IsService() || m.Action == nil {
		return nil
	}
	self := botSelfID()
	if self == 0 {
		return nil
	}
	kind := strings.ToLower(fmt.Sprintf("%T %v", m.Action, m.Action))
	hasBot := strings.Contains(kind, fmt.Sprintf("%d", self))
	added := strings.Contains(kind, "adduser") || strings.Contains(kind, "joined")
	kicked := strings.Contains(kind, "deleteuser") || strings.Contains(kind, "kick")
	if !hasBot && userIDOf(m) != self {
		return nil
	}
	if !added && !kicked {
		return nil
	}
	log.Println("service membership", kind)
	chatID := canonChatID(m.ChatID())
	if added {
		addServedChat(chatID)
		addBroadcastChat(chatID, "group")
	}
	go logBotMembership(chatID, userIDOf(m), senderFullName(m), senderUsername(m), kicked)
	return nil
}

func handleRawChannelParticipant(u telegram.Update, c *telegram.Client) error {
	upd, ok := u.(*telegram.UpdateChannelParticipant)
	if !ok || upd == nil {
		return nil
	}
	self := botSelfID()
	if self == 0 || upd.UserID != self {
		return nil
	}
	chatID := canonChatID(-1000000000000 - upd.ChannelID)
	newName := strings.ToLower(fmt.Sprintf("%T", upd.NewParticipant))
	kicked := strings.Contains(newName, "banned") || strings.Contains(newName, "left") || upd.NewParticipant == nil
	added := !kicked
	log.Println("raw channel participant added=", added, "kicked=", kicked, "new=", newName)
	byName, byUser, byID := "User", "-", upd.ActorID
	if added {
		addServedChat(chatID)
		addBroadcastChat(chatID, "group")
	}
	go logBotMembership(chatID, byID, byName, byUser, kicked)
	return nil
}
