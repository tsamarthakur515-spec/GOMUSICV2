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
		{styleURLBtn("+ лдд мє тσ уσур грσуп", BotLink+"?startgroup=true", ColourRed)},
		{styleBtn("нєлп & сσммандs", "show_help", ColourGreen)},
		{styleURLBtn("упдатєs", UpdatesChannel, ColourBlue), styleURLBtn("sуппσрт", SupportGroup, ColourGreen)},
		{styleURLBtn("sσурсє сσдє", ownerURL(), ColourRed)},
	}
}

func GetAboutMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn(" васк", "go_back", ColourRed)},
	}
}

func GetHelpMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn("адмɪη", "help_admin", ColourRed), styleBtn("аутн", "help_auth", ColourBlue), styleBtn("в-саsт", "help_gcast", ColourGreen)},
		{styleBtn("плау", "help_play", ColourRed), styleBtn("sудσ", "help_sudo", ColourBlue), styleBtn("рєsтрɪст", "help_restrict", ColourGreen)},
		{styleBtn("sтарт", "help_start", ColourRed), styleBtn("аутσплау", "help_autoplay", ColourBlue), styleBtn("ɪηлɪηє", "help_inline", ColourGreen)},
		{styleBtn(" васк", "go_back", ColourRed)},
	}
}

func GetHelpHomeMarkup() [][]InlineBtn { return GetHelpMarkup() }

func GetBackMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn(" васк", "show_help", ColourRed)},
	}
}

func GetGroupStartMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleURLBtn(" сσммандs", BotLink+"?start=pm_help", ColourGreen)},
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
		{styleBtn("слσsє", "close_panel", ColourRed)},
	}
}

func GetQueuedMarkup(chatID int64, index int) [][]InlineBtn {
	return [][]InlineBtn{
		{
			styleBtn("▷ плау ησṡ", fmt.Sprintf("queue_now:%d:%d", chatID, index), ColourBlue),
			styleBtn("‣‣I sкɪп", fmt.Sprintf("skip:%d", chatID), ColourGreen),
		},
		{styleBtn("слσsє", "close_panel", ColourRed)},
	}
}
