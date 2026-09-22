package main

import (
	"fmt"
	"strings"
)

const (
	ColourRed   = "red"
	ColourBlue  = "blue"
	ColourGreen = "green"
)

type InlineBtn struct {
	Text   string
	Data   string
	URL    string
	Colour string
}

func colourToStyle(colour string) string {
	switch strings.ToLower(colour) {
	case ColourRed:
		return "danger"
	case ColourGreen:
		return "success"
	case ColourBlue:
		return "primary"
	default:
		return ""
	}
}

func urlBtn(text, url string) InlineBtn {
	return InlineBtn{Text: text, URL: url}
}

func dataBtn(text, cb string) InlineBtn {
	return InlineBtn{Text: text, Data: cb}
}

func styleBtn(text, cb, colour string) InlineBtn {
	b := dataBtn(text, cb)
	b.Colour = colour
	return b
}

func styleURLBtn(text, url, colour string) InlineBtn {
	b := urlBtn(text, url)
	b.Colour = colour
	return b
}

func DataBtn(text, cb string) InlineBtn {
	return dataBtn(text, cb)
}

func UrlBtn(text, url string) InlineBtn {
	return urlBtn(text, url)
}

func GetStartMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleURLBtn("+ "+smallcaps("add me in your group")+" +", BotLink+"?startgroup=true", ColourRed)},
		{styleURLBtn(smallcaps("owner"), fmt.Sprintf("tg://user?id=%d", OwnerID), ColourRed), styleBtn(smallcaps("about"), "about_menu", ColourRed)},
		{styleURLBtn(smallcaps("support"), SupportGroup, ColourGreen), styleURLBtn(smallcaps("update"), UpdatesChannel, ColourGreen)},
		{styleBtn(smallcaps("help and commands"), "show_help", ColourGreen)},
	}
}

func GetAboutMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn(smallcaps("back"), "go_back", ColourRed)},
	}
}

func GetHelpMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn(smallcaps("admin"), "help_admin", ColourRed), styleBtn(smallcaps("autoplay"), "help_autoplay", ColourBlue), styleBtn(smallcaps("gcast"), "help_gcast", ColourGreen)},
		{styleBtn(smallcaps("bl-chat"), "help_blchat", ColourRed), styleBtn(smallcaps("bl-users"), "help_blusers", ColourBlue), styleBtn(smallcaps("ping"), "help_ping", ColourGreen)},
		{styleBtn(smallcaps("play"), "help_play", ColourRed), styleBtn(smallcaps("speed"), "help_speed", ColourBlue), styleBtn(smallcaps("info"), "help_info", ColourGreen)},
		{styleBtn(smallcaps("close"), "close_help", ColourRed)},
	}
}

func GetHelpHomeMarkup() [][]InlineBtn {
	rows := GetHelpMarkup()
	rows[len(rows)-1] = []InlineBtn{styleBtn(smallcaps("back"), "go_back", ColourRed)}
	return rows
}

func GetBackMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn(smallcaps("back"), "show_help", ColourRed)},
		{styleBtn(smallcaps("close"), "close_help", ColourRed)},
	}
}

func GetGroupStartMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn(smallcaps("help"), "show_help", ColourGreen), styleBtn(smallcaps("about"), "about_menu", ColourRed)},
	}
}

func GetNowPlayingMarkup(bar string) [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn("▷", "resume", ColourRed), styleBtn("II", "pause", ColourBlue), styleBtn("‣‣I", "skip", ColourGreen), styleBtn("▢", "stop", ColourRed)},
		{styleBtn("-10s", "seek_back", ColourBlue), styleBtn("+10s", "seek_fwd", ColourGreen)},
		{styleBtn(bar, "progress", ColourGreen)},
		{styleBtn("Close", "close_panel", ColourRed)},
	}
}
