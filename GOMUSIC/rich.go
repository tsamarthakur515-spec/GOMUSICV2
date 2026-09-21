package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"
	"unicode/utf16"

	"github.com/amarnathcjd/gogram/telegram"
)

var (
	reImgSrc = regexp.MustCompile(`(?is)<img[^>]*src=["']([^"']+)["'][^>]*>`)
	reTgBtn  = regexp.MustCompile(`(?is)<tg-button[^>]*>.*?</tg-button>`)
	menuPic  sync.Map
)

func menuPicKey(chatID int64, msgID int32) string {
	return fmt.Sprintf("%d:%d", chatID, msgID)
}

func rememberMenuPic(chatID int64, msgID int32, photo string) {
	if photo != "" && chatID != 0 && msgID != 0 {
		menuPic.Store(menuPicKey(chatID, msgID), photo)
	}
}

func lookupMenuPic(chatID int64, msgID int32, msg *telegram.NewMessage) any {
	if msg != nil {
		if p := msg.Photo(); p != nil {
			return p
		}
		if m := msg.Media(); m != nil {
			return m
		}
	}
	if v, ok := menuPic.Load(menuPicKey(chatID, msgID)); ok {
		return v
	}
	if len(StartPhotos) > 0 {
		return StartPhotos[0]
	}
	return nil
}

func richEsc(v string) string { return html.EscapeString(v) }

func sanitizeDisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "User"
	}
	return name
}

func richHeading(text string, level int) string {
	_ = level
	return "<b>" + text + "</b>\n"
}

func richNote(text string) string {
	text = strings.ReplaceAll(text, "<p>", "")
	text = strings.ReplaceAll(text, "</p>", "\n")
	return strings.TrimSpace(text) + "\n"
}

func richImg(url string) string {
	if url == "" {
		return ""
	}
	return `<img src="` + html.EscapeString(url) + `">`
}

func richTable(headers []string, rows [][]string) string {
	var b strings.Builder
	for _, row := range rows {
		if len(row) >= 2 {
			b.WriteString(row[0] + " — " + row[1] + "\n")
		} else if len(row) == 1 {
			b.WriteString(row[0] + "\n")
		}
	}
	_ = headers
	return b.String()
}

func richKVTable(pairs [][2]string) string {
	var b strings.Builder
	for _, p := range pairs {
		if p[1] == "" {
			continue
		}
		b.WriteString("<b>" + p[0] + ":</b> " + p[1] + "\n")
	}
	return b.String()
}

func richDetails(summary, body string, open bool) string {
	_ = open
	body = strings.ReplaceAll(body, "<p>", "")
	body = strings.ReplaceAll(body, "</p>", "\n")
	return "<b>" + summary + "</b>\n" + strings.TrimSpace(body) + "\n"
}

func supportUpdatesPills() string { return "" }

func extractPhoto(content string) (photo string, text string) {
	if m := reImgSrc.FindStringSubmatch(content); len(m) == 2 {
		photo = html.UnescapeString(m[1])
		content = reImgSrc.ReplaceAllString(content, "")
	}
	return photo, content
}

func telegramHTML(s string) string {
	s = reTgBtn.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = strings.ReplaceAll(s, "</p>", "\n")
	s = strings.ReplaceAll(s, "<p>", "")
	s = strings.ReplaceAll(s, "</h1>", "\n")
	s = strings.ReplaceAll(s, "</h2>", "\n")
	s = strings.ReplaceAll(s, "</h3>", "\n")
	s = strings.ReplaceAll(s, "</h4>", "\n")
	s = strings.ReplaceAll(s, "<h1>", "<b>")
	s = strings.ReplaceAll(s, "<h2>", "<b>")
	s = strings.ReplaceAll(s, "<h3>", "<b>")
	s = strings.ReplaceAll(s, "<h4>", "<b>")
	s = strings.ReplaceAll(s, "</table>", "\n")
	s = strings.ReplaceAll(s, "</tr>", "\n")
	s = strings.ReplaceAll(s, "</td>", " ")
	s = strings.ReplaceAll(s, "</th>", " ")
	s = strings.ReplaceAll(s, "<table border=\"1\">", "")
	s = strings.ReplaceAll(s, "<table>", "")
	s = strings.ReplaceAll(s, "<tr>", "")
	s = strings.ReplaceAll(s, "<td>", "")
	s = strings.ReplaceAll(s, "<th>", "")
	s = strings.ReplaceAll(s, "<details open>", "")
	s = strings.ReplaceAll(s, "<details>", "")
	s = strings.ReplaceAll(s, "</details>", "\n")
	s = strings.ReplaceAll(s, "<summary>", "<b>")
	s = strings.ReplaceAll(s, "</summary>", "</b>\n")
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(s)
}

func utf16Count(s string) int { return len(utf16.Encode([]rune(s))) }

func htmlToPlain(s string) string {
	_, plain := Bot.FormatMessage(telegramHTML(s), "HTML")
	return plain
}

func isNotModified(err error) bool {
	if err == nil {
		return false
	}
	low := strings.ToLower(err.Error())
	return strings.Contains(low, "not modified") || strings.Contains(low, "message_not_modified")
}

func htmlSendOpts(markup telegram.ReplyMarkup) *telegram.SendOptions {
	return &telegram.SendOptions{ParseMode: "HTML", ReplyMarkup: markup}
}

func htmlMediaOpts(caption string, markup telegram.ReplyMarkup) *telegram.MediaOptions {
	return &telegram.MediaOptions{Caption: caption, ParseMode: "HTML", ReplyMarkup: markup}
}

func msgBotChatID(msg *telegram.NewMessage) int64 {
	if msg == nil {
		return 0
	}
	if id := msg.ChannelID(); id != 0 {
		return id
	}
	return msg.ChatID()
}

func hasBlockquote(ents []telegram.MessageEntity) bool {
	for _, e := range ents {
		if _, ok := e.(*telegram.MessageEntityBlockquote); ok {
			return true
		}
	}
	return false
}

func captionEntities(htmlText string) ([]telegram.MessageEntity, string) {
	text := telegramHTML(htmlText)
	ents, plain := Bot.FormatMessage(text, "HTML")
	if !hasBlockquote(ents) && strings.TrimSpace(plain) != "" {
		ents = append([]telegram.MessageEntity{&telegram.MessageEntityBlockquote{
			Collapsed: true,
			Offset:    0,
			Length:    int32(utf16Count(plain)),
		}}, ents...)
	}
	return ents, plain
}

func editPhotoCaption(chatID int64, msgID int32, msg *telegram.NewMessage, htmlText string, markup telegram.ReplyMarkup) error {
	htmlText = telegramHTML(htmlText)
	ents, _ := captionEntities(htmlText)
	opts := &telegram.SendOptions{
		ParseMode:   "HTML",
		Entities:    ents,
		ReplyMarkup: markup,
		Media:       lookupMenuPic(chatID, msgID, msg),
	}
	_, err := Bot.EditMessage(chatID, msgID, htmlText, opts)
	if isNotModified(err) {
		return nil
	}
	return err
}

func sendHTML(client *telegram.Client, chat any, content string, markup telegram.ReplyMarkup) (*telegram.NewMessage, error) {
	photo, raw := extractPhoto(content)
	text := telegramHTML(raw)
	if photo != "" {
		msg, err := client.SendMedia(chat, photo, htmlMediaOpts(text, markup))
		if err == nil && msg != nil {
			rememberMenuPic(msgBotChatID(msg), msg.ID, photo)
			return msg, nil
		}
	}
	msg, err := client.SendMessage(chat, text, htmlSendOpts(markup))
	if err != nil {
		return client.SendMessage(chat, htmlToPlain(text), &telegram.SendOptions{ReplyMarkup: markup})
	}
	return msg, nil
}

func editHTML(msg *telegram.NewMessage, content string, markup telegram.ReplyMarkup) error {
	if msg == nil {
		return nil
	}
	photo, raw := extractPhoto(content)
	if photo != "" {
		rememberMenuPic(msgBotChatID(msg), msg.ID, photo)
	}
	return editPhotoCaption(msgBotChatID(msg), msg.ID, msg, raw, markup)
}

func editMenu(cb *telegram.CallbackQuery, content string, markup telegram.ReplyMarkup) error {
	if cb == nil {
		return nil
	}
	msg, _ := cb.GetMessage()
	if msg == nil {
		return nil
	}
	photo, raw := extractPhoto(content)
	chatID := cb.ChatID
	if chatID == 0 {
		chatID = msgBotChatID(msg)
	}
	if photo != "" {
		rememberMenuPic(chatID, msg.ID, photo)
	}
	return editPhotoCaption(chatID, msg.ID, msg, raw, markup)
}

func mixedKeyboard(rows [][][2]string) telegram.ReplyMarkup {
	kb := telegram.NewKeyboard()
	for _, row := range rows {
		btns := make([]telegram.KeyboardInlineButton, 0, len(row))
		for _, b := range row {
			if strings.HasPrefix(b[1], "http://") || strings.HasPrefix(b[1], "https://") || strings.HasPrefix(b[1], "tg://") {
				btns = append(btns, telegram.Button.URL(b[0], b[1]))
			} else {
				btns = append(btns, telegram.Button.Data(b[0], b[1]))
			}
		}
		kb.AddRow(btns...)
	}
	return kb.Build()
}
