package main

import (
	"log"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func styleFromColour(colour string) *telegram.KeyboardButtonStyle {
	switch strings.ToLower(colour) {
	case ColourRed:
		return &telegram.KeyboardButtonStyle{BgDanger: true}
	case ColourGreen:
		return &telegram.KeyboardButtonStyle{BgSuccess: true}
	default:
		return &telegram.KeyboardButtonStyle{BgPrimary: true}
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
			btn.Style = styleFromColour(b.Colour)
			btns = append(btns, btn)
		}
		if len(btns) > 0 {
			kb.AddRow(btns...)
		}
	}
	return kb.Build()
}

func quotedEntities(inner string) ([]telegram.MessageEntity, string) {
	caption := "<blockquote>" + strings.TrimSpace(inner) + "</blockquote>"
	ents, plain := Bot.FormatMessage(caption, "HTML")
	hasBQ := false
	for _, e := range ents {
		if bq, ok := e.(*telegram.MessageEntityBlockquote); ok {
			bq.Collapsed = false
			hasBQ = true
		}
	}
	if !hasBQ && strings.TrimSpace(plain) != "" {
		ents = append(ents, &telegram.MessageEntityBlockquote{
			Collapsed: false,
			Offset:    0,
			Length:    int32(utf16Count(plain)),
		})
	}
	return ents, plain
}

func sendQuotedPhoto(chatID int64, inner string, rows [][]InlineBtn) (*telegram.NewMessage, error) {
	ents, plain := quotedEntities(inner)
	photo := pickStartPhoto()
	markup := gogramMarkup(rows)
	msg, err := Bot.SendMedia(chatID, photo, &telegram.MediaOptions{
		Caption:     plain,
		ParseMode:   "",
		Entities:    ents,
		ReplyMarkup: markup,
	})
	if err != nil {
		log.Println("sendQuotedPhoto:", err)
	}
	return msg, err
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
	ents, plain := quotedEntities(inner)
	markup := gogramMarkup(rows)
	photo := pickStartPhoto()
	_, err := Bot.EditMessage(chatID, msgID, plain, &telegram.SendOptions{
		Entities:    ents,
		Media:       &telegram.InputMediaPhotoExternal{URL: photo},
		ReplyMarkup: markup,
	})
	if err == nil || isNotModified(err) {
		return
	}
	log.Println("showQuotedMenu edit:", err)
	_, _ = sendQuotedPhoto(chatID, inner, rows)
	if msg != nil {
		_, _ = msg.Delete()
	}
}
