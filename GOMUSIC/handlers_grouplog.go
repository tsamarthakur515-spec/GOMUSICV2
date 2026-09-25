package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func profileURL(uid int64, username string) string {
	u := strings.TrimPrefix(strings.TrimSpace(username), "@")
	if u != "" && u != "-" {
		return "https://t.me/" + u
	}
	if uid != 0 {
		return fmt.Sprintf("tg://user?id=%d", uid)
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

func groupTypeAndLink(chatID int64) (gtype, link, title string) {
	gtype = smallcaps("private")
	link = "-"
	title = "Group"
	if Bot == nil {
		return
	}
	chat, err := Bot.GetChat(chatID)
	if err != nil || chat == nil {
		return
	}
	if strings.TrimSpace(chat.Title) != "" {
		title = chat.Title
	}
	if uname := strings.TrimSpace(chat.Username); uname != "" {
		gtype = smallcaps("public")
		link = "https://t.me/" + uname
	}
	return
}

func logBotGroupEvent(m *telegram.NewMessage, kicked bool) {
	if LoggerID == 0 || Bot == nil || m == nil || m.IsPrivate() {
		return
	}
	chatID := m.ChatID()
	if chatID == 0 || chatID == LoggerID {
		return
	}
	uid := userIDOf(m)
	name := senderFullName(m)
	uname := senderUsername(m)
	gtype, link, title := groupTypeAndLink(chatID)
	head := smallcaps("bot added in new group")
	if kicked {
		head = smallcaps("bot was kick from group")
	}
	body := "<blockquote expandable><b>" + head + "</b>\n\n" +
		smallcaps("title") + " : " + richEsc(title) + "\n" +
		smallcaps("name") + " : " + mentionHTML(uid, name) + "\n" +
		smallcaps("u name") + " : <code>" + richEsc(uname) + "</code>\n" +
		smallcaps("u id") + " : <code>" + fmt.Sprintf("%d", uid) + "</code>\n" +
		smallcaps("g name") + " : " + richEsc(title) + "\n" +
		smallcaps("g id") + " : <code>" + fmt.Sprintf("%d", chatID) + "</code>\n" +
		smallcaps("g type") + " : " + gtype
	if link != "-" {
		body += "\n" + smallcaps("link") + " : <a href=\"" + richEsc(link) + "\">" + richEsc(link) + "</a>"
	}
	body += "</blockquote>"
	sendLogger(body, profileMarkup(uid, uname))
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

func actionHasBot(ids []int64, self int64) bool {
	if self == 0 {
		return false
	}
	for _, id := range ids {
		if id == self {
			return true
		}
	}
	return false
}

func handleServiceMessage(m *telegram.NewMessage) error {
	if m == nil || m.Message == nil || m.IsPrivate() {
		return nil
	}
	act := m.Message.Action
	if act == nil {
		return nil
	}
	self := botSelfID()
	switch a := act.(type) {
	case *telegram.MessageActionChatAddUser:
		if actionHasBot(a.Users, self) {
			addServedChat(m.ChatID())
			addBroadcastChat(m.ChatID(), "group")
			go logBotGroupEvent(m, false)
		}
	case *telegram.MessageActionChatJoinedByLink:
		if self != 0 && userIDOf(m) == self {
			addServedChat(m.ChatID())
			addBroadcastChat(m.ChatID(), "group")
			go logBotGroupEvent(m, false)
		}
	case *telegram.MessageActionChatDeleteUser:
		if self != 0 && a.UserID == self {
			go logBotGroupEvent(m, true)
		}
	default:
		name := strings.ToLower(fmt.Sprintf("%T", act))
		if strings.Contains(name, "adduser") || strings.Contains(name, "joined") {
			log.Println("group service action", name)
		}
	}
	return nil
}
