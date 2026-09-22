package main

import (
	"fmt"
	"html"
	"reflect"
	"regexp"
	"sort"
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

func menuPhotoURL(chatID int64, msgID int32) string {
	if v, ok := menuPic.Load(menuPicKey(chatID, msgID)); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func withMenuPhoto(content string, chatID int64, msgID int32) string {
	if p, _ := extractPhoto(content); p != "" {
		return content
	}
	if url := menuPhotoURL(chatID, msgID); url != "" {
		return richImg(url) + content
	}
	return content
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

func entityBounds(entity telegram.MessageEntity) (offset, length int32) {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.IsValid() && v.Kind() == reflect.Struct {
		off := v.FieldByName("Offset")
		lengthField := v.FieldByName("Length")
		if off.IsValid() && lengthField.IsValid() {
			return int32(off.Int()), int32(lengthField.Int())
		}
	}
	return 0, 0
}

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
	sort.SliceStable(ents, func(i, j int) bool {
		offI, lenI := entityBounds(ents[i])
		offJ, lenJ := entityBounds(ents[j])
		if offI != offJ {
			return offI < offJ
		}
		return lenI > lenJ
	})
	return ents, plain
}

func editPhotoCaption(chatID int64, msgID int32, msg *telegram.NewMessage, htmlText string, markup telegram.ReplyMarkup) error {
	ents, plain := captionEntities(htmlText)
	_, err := Bot.EditMessage(chatID, msgID, plain, &telegram.SendOptions{
		ParseMode:   "",
		Entities:    ents,
		ReplyMarkup: markup,
	})
	if isNotModified(err) {
		return nil
	}
	return err
}

func sendHTML(client *telegram.Client, chat any, content string, markup telegram.ReplyMarkup) (*telegram.NewMessage, error) {
	photo, raw := extractPhoto(content)
	text := telegramHTML(raw)
	ents, plain := captionEntities(text)
	if photo != "" {
		opts := &telegram.MediaOptions{
			Caption:     plain,
			ParseMode:   "",
			Entities:    ents,
			ReplyMarkup: markup,
		}
		msg, err := client.SendMedia(chat, photo, opts)
		if err == nil && msg != nil {
			rememberMenuPic(msgBotChatID(msg), msg.ID, photo)
			return msg, nil
		}
	}
	msg, err := client.SendMessage(chat, plain, &telegram.SendOptions{
		ParseMode:   "",
		Entities:    ents,
		ReplyMarkup: markup,
	})
	if err != nil {
		return client.SendMessage(chat, plain, &telegram.SendOptions{ReplyMarkup: markup})
	}
	return msg, nil
}

func editHTML(msg *telegram.NewMessage, content string, markup telegram.ReplyMarkup) error {
	if msg == nil {
		return nil
	}
	_, raw := extractPhoto(content)
	return editPhotoCaption(msgBotChatID(msg), msg.ID, msg, raw, markup)
}

func editMenu(cb *telegram.CallbackQuery, content string, markup telegram.ReplyMarkup) error {
	if cb == nil {
		return nil
	}
	msg, err := cb.GetMessage()
	if err != nil || msg == nil {
		return err
	}
	chatID := cb.ChatID
	if chatID == 0 {
		chatID = msgBotChatID(msg)
	}
	_, raw := extractPhoto(content)
	if photoURL := menuPhotoURL(chatID, msg.ID); photoURL != "" {
		ents, plain := captionEntities(raw)
		_, err = Bot.EditMessage(chatID, msg.ID, plain, &telegram.SendOptions{
			ParseMode:   "",
			Entities:    ents,
			Media:       &telegram.InputMediaPhotoExternal{URL: photoURL},
			ReplyMarkup: markup,
		})
	} else {
		err = editPhotoCaption(chatID, msg.ID, msg, raw, markup)
	}
	if isNotModified(err) {
		return nil
	}
	return err
}

func styleMenuButton(button telegram.KeyboardInlineButton, colour string) telegram.KeyboardInlineButton {
	switch strings.ToLower(colour) {
	case "red":
		button.Style = &telegram.KeyboardButtonStyle{BgDanger: true}
	case "green":
		button.Style = &telegram.KeyboardButtonStyle{BgSuccess: true}
	default:
		button.Style = &telegram.KeyboardButtonStyle{BgPrimary: true}
	}
	return button
}

func mixedKeyboard(rows [][][2]string) telegram.ReplyMarkup {
	kb := telegram.NewKeyboard()
	for rowIndex, row := range rows {
		btns := make([]telegram.KeyboardInlineButton, 0, len(row))
		for buttonIndex, b := range row {
			colour := "blue"
			switch (rowIndex + buttonIndex) % 3 {
			case 0:
				colour = "red"
			case 2:
				colour = "green"
			}
			var button telegram.KeyboardInlineButton
			if strings.HasPrefix(b[1], "http://") || strings.HasPrefix(b[1], "https://") || strings.HasPrefix(b[1], "tg://") {
				button = telegram.Button.URL(b[0], b[1])
			} else {
				button = telegram.Button.Data(b[0], b[1])
			}
			btns = append(btns, styleMenuButton(button, colour))
		}
		kb.AddRow(btns...)
	}
	return kb.Build()
}
