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
	plain, _ := htmlToEntities(s)
	return plain
}

func utf16Count(s string) int {
	return len(utf16.Encode([]rune(s)))
}

func htmlToEntities(raw string) (string, []map[string]any) {
	s := telegramHTML(raw)
	var plain strings.Builder
	type openTag struct {
		kind string
		off  int
		url  string
	}
	var stack []openTag
	var ents []map[string]any
	i := 0
	for i < len(s) {
		if s[i] != '<' {
			r, size := decodeRune(s[i:])
			plain.WriteRune(r)
			i += size
			continue
		}
		end := strings.IndexByte(s[i:], '>')
		if end < 0 {
			plain.WriteString(s[i:])
			break
		}
		tag := strings.TrimSpace(s[i+1 : i+end])
		i = i + end + 1
		low := strings.ToLower(tag)
		closing := strings.HasPrefix(low, "/")
		if closing {
			low = strings.TrimSpace(low[1:])
		}
		name := strings.Fields(low)
		if len(name) == 0 {
			continue
		}
		kind := name[0]
		switch kind {
		case "blockquote":
			if closing {
				for n := len(stack) - 1; n >= 0; n-- {
					if stack[n].kind == "blockquote" || stack[n].kind == "expandable_blockquote" {
						e := stack[n]
						stack = append(stack[:n], stack[n+1:]...)
						ents = append(ents, map[string]any{"type": e.kind, "offset": e.off, "length": utf16Count(plain.String()) - e.off})
						break
					}
				}
				continue
			}
			k := "blockquote"
			if strings.Contains(low, "expandable") {
				k = "expandable_blockquote"
			}
			stack = append(stack, openTag{kind: k, off: utf16Count(plain.String())})
		case "a":
			if closing {
				for n := len(stack) - 1; n >= 0; n-- {
					if stack[n].kind == "text_link" {
						e := stack[n]
						stack = append(stack[:n], stack[n+1:]...)
						item := map[string]any{"type": "text_link", "offset": e.off, "length": utf16Count(plain.String()) - e.off}
						if e.url != "" {
							item["url"] = e.url
						}
						ents = append(ents, item)
						break
					}
				}
				continue
			}
			url := hrefOf(tag)
			stack = append(stack, openTag{kind: "text_link", off: utf16Count(plain.String()), url: url})
		case "b", "strong":
			if closing {
				closeSimple(&stack, &ents, "bold", utf16Count(plain.String()))
			} else {
				stack = append(stack, openTag{kind: "bold", off: utf16Count(plain.String())})
			}
		case "i", "em":
			if closing {
				closeSimple(&stack, &ents, "italic", utf16Count(plain.String()))
			} else {
				stack = append(stack, openTag{kind: "italic", off: utf16Count(plain.String())})
			}
		case "code":
			if closing {
				closeSimple(&stack, &ents, "code", utf16Count(plain.String()))
			} else {
				stack = append(stack, openTag{kind: "code", off: utf16Count(plain.String())})
			}
		case "u":
			if closing {
				closeSimple(&stack, &ents, "underline", utf16Count(plain.String()))
			} else {
				stack = append(stack, openTag{kind: "underline", off: utf16Count(plain.String())})
			}
		}
	}
	for n := len(stack) - 1; n >= 0; n-- {
		e := stack[n]
		item := map[string]any{"type": e.kind, "offset": e.off, "length": utf16Count(plain.String()) - e.off}
		if e.url != "" {
			item["url"] = e.url
		}
		ents = append(ents, item)
	}
	out := html.UnescapeString(plain.String())
	if !hasEntityType(ents, "expandable_blockquote") && !hasEntityType(ents, "blockquote") && strings.TrimSpace(out) != "" {
		ents = append([]map[string]any{{"type": "expandable_blockquote", "offset": 0, "length": utf16Count(out)}}, ents...)
	}
	return out, ents
}

func decodeRune(s string) (rune, int) {
	if s == "" {
		return 0, 0
	}
	r := []rune(s)
	return r[0], len(string(r[0]))
}

func hrefOf(tag string) string {
	low := strings.ToLower(tag)
	for _, key := range []string{`href="`, `href='`, `href=`} {
		idx := strings.Index(low, key)
		if idx < 0 {
			continue
		}
		start := idx + len(key)
		rest := tag[start:]
		switch {
		case strings.HasPrefix(key, `href="`):
			if n := strings.Index(rest, `"`); n >= 0 {
				return html.UnescapeString(rest[:n])
			}
		case strings.HasPrefix(key, `href='`):
			if n := strings.Index(rest, `'`); n >= 0 {
				return html.UnescapeString(rest[:n])
			}
		default:
			f := strings.Fields(rest)
			if len(f) > 0 {
				return html.UnescapeString(strings.Trim(f[0], `"'`))
			}
		}
	}
	return ""
}

func closeSimple(stack *[]openTagLite, ents *[]map[string]any, kind string, end int) {
}

type openTagLite struct {
	kind string
	off  int
	url  string
}

func closeKind(stack *[]struct {
	kind string
	off  int
	url  string
}, ents *[]map[string]any, kind string, end int) {
	s := *stack
	for n := len(s) - 1; n >= 0; n-- {
		if s[n].kind == kind {
			*ents = append(*ents, map[string]any{"type": kind, "offset": s[n].off, "length": end - s[n].off})
			*stack = append(s[:n], s[n+1:]...)
			return
		}
	}
}

func hasEntityType(ents []map[string]any, typ string) bool {
	for _, e := range ents {
		if e["type"] == typ {
			return true
		}
	}
	return false
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
	plain, ents := htmlToEntities(htmlText)
	method := "editMessageText"
	body := map[string]any{
		"chat_id":    chatID,
		"message_id": msgID,
	}
	if caption {
		method = "editMessageCaption"
		body["caption"] = plain
		if len(ents) > 0 {
			body["caption_entities"] = ents
		}
	} else {
		body["text"] = plain
		body["disable_web_page_preview"] = true
		if len(ents) > 0 {
			body["entities"] = ents
		}
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
	// Always try caption first: start/help/about/player are photo posts.
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

func editMenu(cb *telegram.CallbackQuery, content string, markup telegram.ReplyMarkup) error {
	if cb == nil {
		return nil
	}
	msg, _ := cb.GetMessage()
	if msg == nil {
		return nil
	}
	_, raw := extractPhoto(content)
	text := telegramHTML(raw)
	chatID := cb.ChatID
	if chatID == 0 {
		chatID = msgBotChatID(msg)
	}
	err := botAPIEdit(chatID, msg.ID, text, markup, true)
	if err == nil || isNotModified(err) {
		return nil
	}
	return botAPIEdit(chatID, msg.ID, text, markup, false)
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
