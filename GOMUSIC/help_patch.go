package main

import (
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func openHelpPage(cb *telegram.CallbackQuery, key string) {
	if key == "" || key == "main" {
		showQuotedMenu(cb, helpMainHTML(), GetHelpHomeMarkup())
		return
	}
	body := helpPage(key)
	rows := GetBackMarkup()
	if !editQuotedCaption(cb, body, rows) {
		showQuotedMenu(cb, body, rows)
	}
}

func helpKeyFromData(data string) string {
	data = strings.TrimSpace(data)
	switch {
	case data == "help_cb", data == "show_help", data == "help:main", data == "help_main":
		return "main"
	case strings.HasPrefix(data, "help:"):
		return strings.TrimPrefix(data, "help:")
	case strings.HasPrefix(data, "help_"):
		return strings.TrimPrefix(data, "help_")
	default:
		return ""
	}
}
