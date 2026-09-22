package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

var menuHTTP = &http.Client{Timeout: 25 * time.Second}

func styledAPIMarkup(rows [][]InlineBtn) map[string]any {
	if len(rows) == 0 {
		return nil
	}
	var out [][]map[string]string
	for _, row := range rows {
		var btns []map[string]string
		for _, b := range row {
			item := map[string]string{"text": b.Text}
			if b.URL != "" {
				item["url"] = b.URL
			} else {
				item["callback_data"] = b.Data
			}
			if b.Colour != "" {
				item["style"] = colourToStyle(b.Colour)
			}
			btns = append(btns, item)
		}
		if len(btns) > 0 {
			out = append(out, btns)
		}
	}
	return map[string]any{"inline_keyboard": out}
}

func botAPICall(method string, body map[string]any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := menuHTTP.Post("https://api.telegram.org/bot"+BotToken+"/"+method, "application/json", bytes.NewReader(raw))
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
	if !parsed.OK {
		desc := parsed.Description
		if desc == "" {
			desc = string(out)
		}
		return fmt.Errorf("%s", desc)
	}
	return nil
}

func startHTMLCaption(inner string) string {
	return "<blockquote expandable>" + strings.TrimSpace(inner) + "</blockquote>"
}

func sendQuotedPhoto(chatID int64, inner string, rows [][]InlineBtn) (*telegram.NewMessage, error) {
	caption := startHTMLCaption(inner)
	photo := pickStartPhoto()
	body := map[string]any{
		"chat_id":    chatID,
		"photo":      photo,
		"caption":    caption,
		"parse_mode": "HTML",
	}
	if kb := styledAPIMarkup(rows); kb != nil {
		body["reply_markup"] = kb
	}
	if err := botAPICall("sendPhoto", body); err == nil {
		return nil, nil
	}
	delete(body, "reply_markup")
	if kb := styledAPIMarkup(rows); kb != nil {
		// retry without style if Telegram rejected button style
		for _, row := range kb["inline_keyboard"].([][]map[string]string) {
			for _, b := range row {
				delete(b, "style")
			}
		}
		body["reply_markup"] = kb
	}
	if err := botAPICall("sendPhoto", body); err == nil {
		return nil, nil
	}
	return Bot.SendMedia(chatID, photo, &telegram.MediaOptions{
		Caption:   caption,
		ParseMode: "HTML",
	})
}

func showQuotedMenu(cb *telegram.CallbackQuery, inner string, rows [][]InlineBtn) {
	if cb == nil {
		return
	}
	chatID := cb.ChatID
	msg, _ := cb.GetMessage()
	if chatID == 0 && msg != nil {
		chatID = msg.ChatID()
	}
	msgID := cb.MessageID
	if msgID == 0 && msg != nil {
		msgID = msg.ID
	}
	caption := startHTMLCaption(inner)
	media := map[string]any{
		"type":       "photo",
		"media":      pickStartPhoto(),
		"caption":    caption,
		"parse_mode": "HTML",
	}
	body := map[string]any{
		"chat_id":    chatID,
		"message_id": msgID,
		"media":      media,
	}
	if kb := styledAPIMarkup(rows); kb != nil {
		body["reply_markup"] = kb
	}
	err := botAPICall("editMessageMedia", body)
	if err == nil || (err != nil && strings.Contains(strings.ToLower(err.Error()), "not modified")) {
		return
	}
	_, _ = sendQuotedPhoto(chatID, inner, rows)
	if msg != nil {
		_, _ = msg.Delete()
	}
}
