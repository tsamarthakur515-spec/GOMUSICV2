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

	"github.com/amarnathcjd/gogram/telegram"
)

var (
	reImgSrc = regexp.MustCompile(`(?is)<img[^>]*src=["']([^"']+)["'][^>]*>`)
	reTgBtn  = regexp.MustCompile(`(?is)<tg-button[^>]*>.*?</tg-button>`)
	botHTTP  = &http.Client{Timeout: 20 * time.Second}
)

func richEsc(v string) string {
	return html.EscapeString(v)
}

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

func supportUpdatesPills() string {
	return ""
}

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

func htmlToPlain(s string) string {
	s = telegramHTML(s)
	s = strings.ReplaceAll(s, "<b>", "")
	s = strings.ReplaceAll(s, "</b>", "")
	s = strings.ReplaceAll(s, "<i>", "")
	s = strings.ReplaceAll(s, "</i>", "")
	s = strings.ReplaceAll(s, "<code>", "")
	s = strings.ReplaceAll(s, "</code>", "")
	for {
		start := strings.Index(s, "<")
		end := strings.Index(s, ">")
		if start < 0 || end < start {
			break
		}
		s = s[:start] + s[end+1:]
	}
	return strings.TrimSpace(html.UnescapeString(s))
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
	return &telegram.MediaOptions{
		Caption:     caption,
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	}
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

func botAPIEdit(chatID int64, msgID int32, htmlText string, markup telegram.ReplyMarkup, caption bool) error {
	if BotToken == "" || chatID == 0 || msgID == 0 {
		return fmt.Errorf("bot api edit missing ids")
	}
	method := "editMessageText"
	body := map[string]any{
		"chat_id":    chatID,
		"message_id": msgID,
		"parse_mode": "HTML",
	}
	if caption {
		method = "editMessageCaption"
		body["caption"] = htmlText
	} else {
		body["text"] = htmlText
		body["disable_web_page_preview"] = true
	}
	if kb := replyMarkupToAPI(markup); kb != nil {
		body["reply_markup"] = kb
	}
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
	text := telegramHTML(raw)
	chatID := msgBotChatID(msg)

	// Start/help/about/player are photo messages. Caption-first like a working
	// edit_caption. Do not MTProto-edit afterwards — that strips blockquote.
	hasMedia := msg.IsMedia() || msg.Photo() != nil || msg.Media() != nil
	if hasMedia {
		err := botAPIEdit(chatID, msg.ID, text, markup, true)
		if err == nil || isNotModified(err) {
			return nil
		}
		err = botAPIEdit(chatID, msg.ID, text, markup, false)
		if err == nil || isNotModified(err) {
			return nil
		}
		return err
	}

	err := botAPIEdit(chatID, msg.ID, text, markup, false)
	if err == nil || isNotModified(err) {
		return nil
	}
	err = botAPIEdit(chatID, msg.ID, text, markup, true)
	if err == nil || isNotModified(err) {
		return nil
	}
	return err
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
