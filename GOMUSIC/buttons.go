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

func DataBtn(text, cb string) InlineBtn { return dataBtn(text, cb) }
func UrlBtn(text, url string) InlineBtn { return urlBtn(text, url) }

func ownerURL() string {
	if OwnerID != 0 {
		return fmt.Sprintf("tg://user?id=%d", OwnerID)
	}
	return SupportGroup
}

func GetStartMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleURLBtn("+ "+smallcaps("add me to your group"), BotLink+"?startgroup=true", ColourRed)},
		{styleBtn(smallcaps("help and commands"), "show_help", ColourGreen)},
		{styleURLBtn(smallcaps("updates"), UpdatesChannel, ColourBlue), styleURLBtn(smallcaps("support"), SupportGroup, ColourGreen)},
		{styleURLBtn(smallcaps("source code"), ownerURL(), ColourRed)},
	}
}

func GetAboutMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn(smallcaps("back"), "go_back", ColourRed)},
	}
}

func GetHelpMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn(smallcaps("admin"), "help_admin", ColourRed), styleBtn(smallcaps("auth"), "help_auth", ColourBlue), styleBtn(smallcaps("bcast"), "help_gcast", ColourGreen)},
		{styleBtn(smallcaps("play"), "help_play", ColourRed), styleBtn(smallcaps("sudo"), "help_sudo", ColourBlue), styleBtn(smallcaps("restrict"), "help_restrict", ColourGreen)},
		{styleBtn(smallcaps("start"), "help_start", ColourRed), styleBtn(smallcaps("autoplay"), "help_autoplay", ColourBlue), styleBtn(smallcaps("inline"), "help_inline", ColourGreen)},
		{styleBtn(smallcaps("back"), "go_back", ColourRed)},
	}
}

func GetHelpHomeMarkup() [][]InlineBtn { return GetHelpMarkup() }

func GetBackMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn(smallcaps("back"), "show_help", ColourRed)},
	}
}

func GetGroupStartMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleURLBtn(smallcaps("commands"), BotLink+"?start=pm_help", ColourGreen)},
	}
}

func GetNowPlayingMarkup(bar string) [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn(bar, "progress", ColourGreen)},
		{
			styleBtn("▷", "resume", ColourRed),
			styleBtn("II", "pause", ColourBlue),
			styleBtn("⟳", "replay", ColourGreen),
			styleBtn("‣‣I", "skip", ColourGreen),
			styleBtn("▢", "stop", ColourRed),
		},
		{styleBtn("-15s", "seek_back", ColourBlue), styleBtn("+15s", "seek_fwd", ColourGreen)},
		{styleBtn(smallcaps("close"), "close_panel", ColourRed)},
	}
}

func GetQueuedMarkup(chatID int64, index int) [][]InlineBtn {
	return [][]InlineBtn{
		{
			styleBtn(smallcaps("play now"), fmt.Sprintf("queue_now:%d:%d", chatID, index), ColourBlue),
			styleBtn(smallcaps("skip"), fmt.Sprintf("skip:%d", chatID), ColourGreen),
		},
		{styleBtn(smallcaps("close"), "close_panel", ColourRed)},
	}
}
