package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

var menuHTTP = &http.Client{Timeout: 25 * time.Second}

var buttonColors = []string{"danger", "primary", "success"}

func randomButtonStyle() string {
	return buttonColors[rand.Intn(len(buttonColors))]
}

func apiMarkup(markup telegram.ReplyMarkup) map[string]any {
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
			item := map[string]string{
				"text":  b.Text,
				"style": randomButtonStyle(),
			}
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

func botAPICall(method string, body map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := menuHTTP.Post("https://api.telegram.org/bot"+BotToken+"/"+method, "application/json", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	var parsed struct {
		OK          bool           `json:"ok"`
		Description string         `json:"description"`
		Result      map[string]any `json:"result"`
	}
	_ = json.Unmarshal(out, &parsed)
	if !parsed.OK {
		desc := parsed.Description
		if desc == "" {
			desc = string(out)
		}
		return nil, fmt.Errorf("%s", desc)
	}
	return parsed.Result, nil
}

func quotedCaption(inner string) string {
	return "<blockquote expandable>" + strings.TrimSpace(inner) + "</blockquote>"
}

func sendQuotedPhoto(chatID int64, inner string, markup telegram.ReplyMarkup) (*telegram.NewMessage, error) {
	photo := pickStartPhoto()
	caption := quotedCaption(inner)
	body := map[string]any{
		"chat_id":    chatID,
		"photo":      photo,
		"caption":    caption,
		"parse_mode": "HTML",
	}
	if kb := apiMarkup(markup); kb != nil {
		body["reply_markup"] = kb
	}
	_, err := botAPICall("sendPhoto", body)
	if err == nil {
		return nil, nil
	}
	return Bot.SendMedia(chatID, photo, &telegram.MediaOptions{
		Caption:     caption,
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	})
}

func showQuotedMenu(cb *telegram.CallbackQuery, inner string, markup telegram.ReplyMarkup) {
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
	photo := pickStartPhoto()
	caption := quotedCaption(inner)
	body := map[string]any{
		"chat_id":    chatID,
		"message_id": msgID,
		"media": map[string]any{
			"type":       "photo",
			"media":      photo,
			"caption":    caption,
			"parse_mode": "HTML",
		},
	}
	if kb := apiMarkup(markup); kb != nil {
		body["reply_markup"] = kb
	}
	_, err := botAPICall("editMessageMedia", body)
	if err == nil || (err != nil && strings.Contains(strings.ToLower(err.Error()), "not modified")) {
		return
	}
	_, _ = sendQuotedPhoto(chatID, inner, markup)
	if msg != nil {
		_, _ = msg.Delete()
	}
}
