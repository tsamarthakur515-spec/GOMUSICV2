package main

import (
	"log"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func editQuotedCaption(cb *telegram.CallbackQuery, inner string, rows [][]InlineBtn) bool {
	if cb == nil {
		return false
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
	if chatID == 0 || msgID == 0 {
		return false
	}
	caption := quotedCaption(inner)
	try := func(withStyle bool) error {
		body := map[string]any{
			"chat_id":    chatID,
			"message_id": msgID,
			"caption":    caption,
			"parse_mode": "HTML",
		}
		if kb := apiMarkup(rows, withStyle); kb != nil {
			body["reply_markup"] = kb
		}
		return botAPICall("editMessageCaption", body)
	}
	if err := try(true); err == nil || (err != nil && strings.Contains(strings.ToLower(err.Error()), "not modified")) {
		return true
	} else {
		log.Println("editMessageCaption styled:", err)
	}
	if err := try(false); err == nil || (err != nil && strings.Contains(strings.ToLower(err.Error()), "not modified")) {
		return true
	} else {
		log.Println("editMessageCaption plain:", err)
	}
	if msg != nil {
		if err := editHTML(msg, caption, gogramMarkup(rows)); err == nil {
			return true
		}
	}
	return false
}
