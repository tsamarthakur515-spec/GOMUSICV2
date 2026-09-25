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

type cachedGroup struct {
	Title  string
	Uname  string
	Invite string
}

type cachedUser struct {
	Name string
	User string
}

var (
	memLogOnce  sync.Map
	groupCache  sync.Map
	userCache   sync.Map
)

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

func rememberUser(id int64, name, uname string) {
	if id == 0 {
		return
	}
	if name == "" || name == "User" {
		return
	}
	userCache.Store(id, cachedUser{Name: name, User: uname})
}

func userFromCache(id int64, name, uname string) (string, string) {
	if name != "" && name != "User" && uname != "" && uname != "-" {
		return name, uname
	}
	if v, ok := userCache.Load(id); ok {
		if u, ok := v.(cachedUser); ok {
			if name == "" || name == "User" {
				name = u.Name
			}
			if uname == "" || uname == "-" {
				uname = u.User
			}
		}
	}
	if name == "" {
		name = "User"
	}
	if uname == "" {
		uname = "-"
	}
	return name, uname
}

func rememberGroup(chatID int64, title, uname, invite string) {
	chatID = canonChatID(chatID)
	if chatID == 0 {
		return
	}
	cur := cachedGroupOf(chatID)
	if title != "" && title != "Group" {
		cur.Title = title
	}
	if uname != "" && uname != "-" {
		cur.Uname = strings.TrimPrefix(uname, "@")
	}
	if invite != "" && invite != "-" && invite != "private" {
		cur.Invite = invite
	}
	groupCache.Store(chatID, cur)
}

func cachedGroupOf(chatID int64) cachedGroup {
	chatID = canonChatID(chatID)
	if v, ok := groupCache.Load(chatID); ok {
		if g, ok := v.(cachedGroup); ok {
			return g
		}
	}
	return cachedGroup{}
}

func exportInvite(chatID int64) string {
	if Bot == nil || chatID == 0 {
		return ""
	}
	peer, err := Bot.ResolvePeer(chatID)
	if err != nil || peer == nil {
		return ""
	}
	res, err := Bot.MessagesExportChatInvite(&telegram.MessagesExportChatInviteParams{Peer: peer})
	if err != nil || res == nil {
		return ""
	}
	switch inv := res.(type) {
	case *telegram.ChatInviteExported:
		return strings.TrimSpace(inv.Link)
	default:
		s := fmt.Sprint(res)
		if i := strings.Index(s, "https://t.me/"); i >= 0 {
			link := s[i:]
			if j := strings.IndexAny(link, " \t\n\"'"); j > 0 {
				link = link[:j]
			}
			return link
		}
	}
	return ""
}

func refreshGroupMeta(chatID int64) cachedGroup {
	chatID = canonChatID(chatID)
	cur := cachedGroupOf(chatID)
	if Bot == nil {
		return cur
	}
	if ch, err := Bot.GetChannel(chatID); err == nil && ch != nil {
		if strings.TrimSpace(ch.Title) != "" {
			cur.Title = ch.Title
		}
		if strings.TrimSpace(ch.Username) != "" {
			cur.Uname = ch.Username
		}
	} else if chat, err := Bot.GetChat(chatID); err == nil && chat != nil {
		if strings.TrimSpace(chat.Title) != "" {
			cur.Title = chat.Title
		}
	}
	if cur.Uname != "" {
		cur.Invite = "https://t.me/" + strings.TrimPrefix(cur.Uname, "@")
	} else if inv := exportInvite(chatID); inv != "" {
		cur.Invite = inv
	}
	groupCache.Store(chatID, cur)
	return cur
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
	rememberUser(uid, name, uname)
	return
}

func logBotMembership(chatID int64, byID int64, byName, byUser string, kicked bool) {
	chatID = canonChatID(chatID)
	if LoggerID == 0 || chatID == 0 || chatID == LoggerID || chatID == canonChatID(LoggerID) {
		return
	}
	byName, byUser = userFromCache(byID, byName, byUser)
	if !kicked && byName == "User" && (byUser == "-" || byUser == "") {
		return
	}
	rememberUser(byID, byName, byUser)
	key := fmt.Sprintf("%d:%v", chatID, kicked)
	if v, ok := memLogOnce.Load(key); ok {
		if t, ok := v.(time.Time); ok && time.Since(t) < 15*time.Second {
			return
		}
	}
	memLogOnce.Store(key, time.Now())

	meta := refreshGroupMeta(chatID)
	if meta.Title == "" {
		meta = cachedGroupOf(chatID)
	}
	title := meta.Title
	if title == "" {
		title = chatTitleOf(chatID)
	}
	if title == "" {
		title = "Unknown Group"
	}
	gtype := smallcaps("private")
	link := meta.Invite
	if meta.Uname != "" {
		gtype = smallcaps("public")
		link = "https://t.me/" + strings.TrimPrefix(meta.Uname, "@")
	}
	if link == "" {
		link = "private"
	}

	head := smallcaps("bot added in new group")
	if kicked {
		head = smallcaps("bot was kick from group")
	}
	body := "<blockquote expandable><b>" + head + "</b>\n\n" +
		smallcaps("name") + " : " + mentionHTML(byID, byName) + "\n" +
		smallcaps("u name") + " : <code>" + richEsc(byUser) + "</code>\n" +
		smallcaps("u id") + " : <code>" + fmt.Sprintf("%d", byID) + "</code>\n" +
		smallcaps("g name") + " : " + richEsc(title) + "\n" +
		smallcaps("g id") + " : <code>" + fmt.Sprintf("%d", chatID) + "</code>\n" +
		smallcaps("g type") + " : " + gtype + "\n" +
		smallcaps("link") + " : "
	if strings.HasPrefix(link, "http") {
		body += "<a href=\"" + richEsc(link) + "\">" + richEsc(link) + "</a>"
	} else {
		body += "<code>" + richEsc(link) + "</code>"
	}
	body += "</blockquote>"
	log.Println("group membership log kicked=", kicked, "chat=", chatID, "title=", title, "by=", byUser)
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
	if p.Channel != nil && strings.TrimSpace(p.Channel.Title) != "" {
		rememberGroup(chatID, p.Channel.Title, p.Channel.Username, "")
	}
	byName, byUser, byID := userBits(p.Actor)
	if byID == 0 {
		byName, byUser, byID = userBits(p.User)
	}
	if added {
		addServedChat(chatID)
		addBroadcastChat(chatID, "group")
		go refreshGroupMeta(chatID)
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
	if m.Channel != nil && strings.TrimSpace(m.Channel.Title) != "" {
		rememberGroup(chatID, m.Channel.Title, m.Channel.Username, "")
	} else if m.Chat != nil && strings.TrimSpace(m.Chat.Title) != "" {
		rememberGroup(chatID, m.Chat.Title, "", "")
	}
	if added {
		addServedChat(chatID)
		addBroadcastChat(chatID, "group")
		go refreshGroupMeta(chatID)
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
	byName, byUser = userFromCache(byID, byName, byUser)
	if added {
		addServedChat(chatID)
		addBroadcastChat(chatID, "group")
		go refreshGroupMeta(chatID)
	}
	go logBotMembership(chatID, byID, byName, byUser, kicked)
	return nil
}
