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
	"unicode/utf16"

	"github.com/amarnathcjd/gogram/telegram"
)

var menuHTTP = &http.Client{Timeout: 25 * time.Second}

var buttonColors = []string{"danger", "primary", "success"}

func randomButtonStyle() string {
	return buttonColors[rand.Intn(len(buttonColors))]
}

func apiMarkup(markup telegram.ReplyMarkup, withStyle bool) map[string]any {
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
			if withStyle {
				item["style"] = randomButtonStyle()
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

func htmlToAPICaption(inner string) (string, []map[string]any) {
	htmlText := "<blockquote expandable>" + strings.TrimSpace(inner) + "</blockquote>"
	ents, plain := Bot.FormatMessage(htmlText, "HTML")
	apiEnts := make([]map[string]any, 0, len(ents)+1)
	hasBQ := false
	for _, e := range ents {
		switch t := e.(type) {
		case *telegram.MessageEntityBlockquote:
			hasBQ = true
			kind := "blockquote"
			if t.Collapsed {
				kind = "expandable_blockquote"
			}
			apiEnts = append(apiEnts, map[string]any{"type": kind, "offset": t.Offset, "length": t.Length})
		case *telegram.MessageEntityBold:
			apiEnts = append(apiEnts, map[string]any{"type": "bold", "offset": t.Offset, "length": t.Length})
		case *telegram.MessageEntityItalic:
			apiEnts = append(apiEnts, map[string]any{"type": "italic", "offset": t.Offset, "length": t.Length})
		case *telegram.MessageEntityCode:
			apiEnts = append(apiEnts, map[string]any{"type": "code", "offset": t.Offset, "length": t.Length})
		case *telegram.MessageEntityTextURL:
			apiEnts = append(apiEnts, map[string]any{"type": "text_link", "offset": t.Offset, "length": t.Length, "url": t.URL})
		case *telegram.MessageEntityMentionName:
			apiEnts = append(apiEnts, map[string]any{"type": "text_link", "offset": t.Offset, "length": t.Length, "url": fmt.Sprintf("tg://user?id=%d", t.UserID)})
		}
	}
	if !hasBQ && strings.TrimSpace(plain) != "" {
		apiEnts = append([]map[string]any{{
			"type":   "expandable_blockquote",
			"offset": 0,
			"length": len(utf16.Encode([]rune(plain))),
		}}, apiEnts...)
	}
	return plain, apiEnts
}

func attachMarkup(body map[string]any, markup telegram.ReplyMarkup, withStyle bool) {
	if kb := apiMarkup(markup, withStyle); kb != nil {
		body["reply_markup"] = kb
	}
}

func sendQuotedPhoto(chatID int64, inner string, markup telegram.ReplyMarkup) (*telegram.NewMessage, error) {
	plain, ents := htmlToAPICaption(inner)
	photo := pickStartPhoto()
	try := func(withStyle bool, useEntities bool) error {
		body := map[string]any{
			"chat_id": chatID,
			"photo":   photo,
			"caption": plain,
		}
		if useEntities && len(ents) > 0 {
			body["caption_entities"] = ents
		} else {
			body["caption"] = "<blockquote expandable>" + strings.TrimSpace(inner) + "</blockquote>"
			body["parse_mode"] = "HTML"
		}
		attachMarkup(body, markup, withStyle)
		_, err := botAPICall("sendPhoto", body)
		return err
	}
	for _, styled := range []bool{true, false} {
		if err := try(styled, true); err == nil {
			return nil, nil
		}
		if err := try(styled, false); err == nil {
			return nil, nil
		}
	}
	caption := "<blockquote expandable>" + strings.TrimSpace(inner) + "</blockquote>"
	msg, err := Bot.SendMedia(chatID, photo, &telegram.MediaOptions{
		Caption:     caption,
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	})
	return msg, err
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
	plain, ents := htmlToAPICaption(inner)
	photo := pickStartPhoto()
	try := func(withStyle bool, useEntities bool) error {
		media := map[string]any{
			"type":  "photo",
			"media": photo,
		}
		if useEntities && len(ents) > 0 {
			media["caption"] = plain
			media["caption_entities"] = ents
		} else {
			media["caption"] = "<blockquote expandable>" + strings.TrimSpace(inner) + "</blockquote>"
			media["parse_mode"] = "HTML"
		}
		body := map[string]any{
			"chat_id":    chatID,
			"message_id": msgID,
			"media":      media,
		}
		attachMarkup(body, markup, withStyle)
		_, err := botAPICall("editMessageMedia", body)
		return err
	}
	for _, styled := range []bool{true, false} {
		if err := try(styled, true); err == nil || (err != nil && strings.Contains(strings.ToLower(err.Error()), "not modified")) {
			return
		}
		if err := try(styled, false); err == nil || (err != nil && strings.Contains(strings.ToLower(err.Error()), "not modified")) {
			return
		}
	}
	_, _ = sendQuotedPhoto(chatID, inner, markup)
	if msg != nil {
		_, _ = msg.Delete()
	}
}
