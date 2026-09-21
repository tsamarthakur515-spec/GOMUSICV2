package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/amarnathcjd/gogram/telegram"
)

var (
	reImgSrc = regexp.MustCompile(`(?is)<img[^>]*src=["']([^"']+)["'][^>]*>`)
	reTgBtn  = regexp.MustCompile(`(?is)<tg-button[^>]*>.*?</tg-button>`)
	botHTTP  = &http.Client{Timeout: 20 * time.Second}
)

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

func replyMarkupToAPI(markup telegram.ReplyMarkup) map[string]any {
	im, ok := markup.(*telegram.ReplyInlineMarkup)
	if !ok || im == nil {
		return nil
	}
	var rows [][]map[string]string
	for _, row := range im.Rows {
		if row == nil {
			continue
		}
		var btns []map[string]string
		for _, b := range row.Buttons {
			if b == nil {
				continue
			}
			item := map[string]string{"text": b.Text}
			switch t := b.Type.(type) {
			case *telegram.InlineButtonTypeCallback:
				item["callback_data"] = string(t.Data)
			case *telegram.InlineButtonTypeURL:
				item["url"] = t.URL
			default:
				continue
			}
			btns = append(btns, item)
		}
		if len(btns) > 0 {
			rows = append(rows, btns)
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return map[string]any{"inline_keyboard": rows}
}

func botAPIPost(method string, body map[string]any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := botHTTP.Post("https://api.telegram.org/bot"+BotToken+"/"+method, "application/json", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	var parsed struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	_ = json.Unmarshal(out, &parsed)
	if parsed.OK {
		return nil
	}
	desc := parsed.Description
	if desc == "" {
		desc = string(out)
	}
	return fmt.Errorf("%s", desc)
}

func editPhotoCaption(chatID int64, msgID int32, msg *telegram.NewMessage, htmlText string, markup telegram.ReplyMarkup) error {
	htmlText = telegramHTML(htmlText)
	ents, plain := captionEntities(htmlText)

	opts := &telegram.SendOptions{
		ParseMode:   "HTML",
		Entities:    ents,
		ReplyMarkup: markup,
	}
	if msg != nil {
		if p := msg.Photo(); p != nil {
			opts.Media = p
		} else if media := msg.Media(); media != nil {
			opts.Media = media
		}
	}
	// Pass HTML string so gogram parseEntities also builds MessageEntityBlockquote.
	_, err := Bot.EditMessage(chatID, msgID, htmlText, opts)
	if err == nil || isNotModified(err) {
		return nil
	}

	// Same photo + HTML caption, like a fresh sendPhoto.
	if msg != nil {
		if fid := msg.FileID(); fid != "" {
			body := map[string]any{
				"chat_id":    chatID,
				"message_id": msgID,
				"media": map[string]any{
					"type":       "photo",
					"media":      fid,
					"caption":    htmlText,
					"parse_mode": "HTML",
				},
			}
			if kb := replyMarkupToAPI(markup); kb != nil {
				body["reply_markup"] = kb
			}
			if e := botAPIPost("editMessageMedia", body); e == nil || isNotModified(e) {
				return nil
			}
		}
	}

	body := map[string]any{
		"chat_id":                  chatID,
		"message_id":               msgID,
		"caption":                  htmlText,
		"parse_mode":               "HTML",
	}
	if kb := replyMarkupToAPI(markup); kb != nil {
		body["reply_markup"] = kb
	}
	if e := botAPIPost("editMessageCaption", body); e == nil || isNotModified(e) {
		return nil
	}
	_, err = Bot.EditMessage(chatID, msgID, plain, &telegram.SendOptions{
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
	if photo != "" {
		msg, err := client.SendMedia(chat, photo, htmlMediaOpts(text, markup))
		if err == nil {
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
	_, raw := extractPhoto(content)
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
	_, raw := extractPhoto(content)
	chatID := cb.ChatID
	if chatID == 0 {
		chatID = msgBotChatID(msg)
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
