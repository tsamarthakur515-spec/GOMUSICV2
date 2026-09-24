package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

var menuHTTP = &http.Client{Timeout: 25 * time.Second}

func styleFromColour(colour string) *telegram.KeyboardButtonStyle {
	switch strings.ToLower(colour) {
	case ColourRed:
		return &telegram.KeyboardButtonStyle{BgDanger: true}
	case ColourGreen:
		return &telegram.KeyboardButtonStyle{BgSuccess: true}
	case ColourBlue:
		return &telegram.KeyboardButtonStyle{BgPrimary: true}
	default:
		return nil
	}
}

func gogramMarkup(rows [][]InlineBtn) telegram.ReplyMarkup {
	kb := telegram.NewKeyboard()
	for _, row := range rows {
		btns := make([]telegram.KeyboardInlineButton, 0, len(row))
		for _, b := range row {
			var btn telegram.KeyboardInlineButton
			if b.URL != "" {
				btn = telegram.Button.URL(b.Text, b.URL)
			} else {
				btn = telegram.Button.Data(b.Text, b.Data)
			}
			if st := styleFromColour(b.Colour); st != nil {
				btn.Style = st
			}
			btns = append(btns, btn)
		}
		if len(btns) > 0 {
			kb.AddRow(btns...)
		}
	}
	return kb.Build()
}

func apiMarkup(rows [][]InlineBtn, withStyle bool) map[string]any {
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
			if withStyle && b.Colour != "" {
				item["style"] = colourToStyle(b.Colour)
			}
			btns = append(btns, item)
		}
		if len(btns) > 0 {
			out = append(out, btns)
		}
	}
	if len(out) == 0 {
		return nil
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

func quotedCaption(inner string) string {
	inner = strings.TrimSpace(inner)
	if strings.Contains(inner, "<blockquote") {
		return inner
	}
	return "<blockquote>" + inner + "</blockquote>"
}

func sendQuotedPhoto(chatID int64, inner string, rows [][]InlineBtn) (*telegram.NewMessage, error) {
	if chatID == 0 {
		return nil, fmt.Errorf("chat id missing")
	}
	caption := quotedCaption(inner)
	photo := pickStartPhoto()
	tryAPI := func(withStyle bool) error {
		body := map[string]any{
			"chat_id":    chatID,
			"photo":      photo,
			"caption":    caption,
			"parse_mode": "HTML",
		}
		if kb := apiMarkup(rows, withStyle); kb != nil {
			body["reply_markup"] = kb
		}
		return botAPICall("sendPhoto", body)
	}
	if err := tryAPI(true); err == nil {
		return nil, nil
	} else {
		log.Println("sendPhoto styled:", err)
	}
	if err := tryAPI(false); err == nil {
		return nil, nil
	} else {
		if strings.Contains(strings.ToLower(err.Error()), "chat not found") {
			log.Println("sendPhoto skipped, chat not found:", chatID)
			return nil, err
		}
		log.Println("sendPhoto plain:", err)
	}
	return Bot.SendMedia(chatID, photo, &telegram.MediaOptions{
		Caption:     caption,
		ParseMode:   "HTML",
		ReplyMarkup: gogramMarkup(rows),
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
	caption := quotedCaption(inner)
	photo := pickStartPhoto()
	tryEdit := func(withStyle bool) error {
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
		if kb := apiMarkup(rows, withStyle); kb != nil {
			body["reply_markup"] = kb
		}
		return botAPICall("editMessageMedia", body)
	}
	if err := tryEdit(true); err == nil || (err != nil && strings.Contains(strings.ToLower(err.Error()), "not modified")) {
		return
	} else {
		log.Println("editMessageMedia styled:", err)
	}
	if err := tryEdit(false); err == nil || (err != nil && strings.Contains(strings.ToLower(err.Error()), "not modified")) {
		return
	} else {
		log.Println("editMessageMedia plain:", err)
	}
	ents, plain := Bot.FormatMessage(caption, "HTML")
	_, err := Bot.EditMessage(chatID, msgID, plain, &telegram.SendOptions{
		Entities:    ents,
		Media:       &telegram.InputMediaPhotoExternal{URL: photo},
		ReplyMarkup: gogramMarkup(rows),
	})
	if err == nil || isNotModified(err) {
		return
	}
	log.Println("gogram edit:", err)
	_, _ = sendQuotedPhoto(chatID, inner, rows)
	if msg != nil {
		_, _ = msg.Delete()
	}
}
