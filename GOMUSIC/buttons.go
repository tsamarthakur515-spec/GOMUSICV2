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
		{styleURLBtn(btnAddMe, BotLink+"?startgroup=true", ColourRed)},
		{styleBtn(btnHelpStart, "help_cb", ColourGreen)},
		{styleURLBtn(btnUpdates, UpdatesChannel, ColourBlue), styleURLBtn(btnSupport, SupportGroup, ColourGreen)},
		{styleURLBtn(btnSource, ownerURL(), ColourRed)},
	}
}

func GetAboutMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn(btnBack, "start", ColourRed)},
	}
}

func GetHelpMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn(btnAdmin, "help:admin", ColourRed), styleBtn(btnAuth, "help:auth", ColourBlue), styleBtn(btnBcast, "help:bcast", ColourGreen)},
		{styleBtn(btnPlay, "help:play", ColourRed), styleBtn(btnSudo, "help:sudo", ColourBlue), styleBtn(btnRestrict, "help:restrict", ColourGreen)},
		{styleBtn(btnThumb, "help:thumb", ColourRed), styleBtn(btnStart, "help:start", ColourBlue), styleBtn(btnAutoplay, "help:autoplay", ColourGreen)},
		{styleBtn(btnPlaylist, "help:playlist", ColourRed), styleBtn(btnVCLogs, "help:vclogs", ColourBlue), styleBtn(btnInline, "help:inline", ColourGreen)},
		{styleBtn(btnBack, "start", ColourRed)},
	}
}

func GetHelpHomeMarkup() [][]InlineBtn { return GetHelpMarkup() }

func GetBackMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn(btnBack, "help:main", ColourRed)},
	}
}

func GetGroupStartMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{styleURLBtn(btnCommands, BotLink+"?start=pm_help", ColourGreen)},
	}
}

func GetNowPlayingMarkup(bar string, autoplayOn bool) [][]InlineBtn {
	return [][]InlineBtn{
		{styleBtn(bar, "progress", ColourGreen)},
		{
			styleBtn("▷", "resume", ColourRed),
			styleBtn("II", "pause", ColourBlue),
			styleBtn("⟳", "replay", ColourGreen),
			styleBtn("‣‣I", "skip", ColourGreen),
			styleBtn("▢", "stop", ColourRed),
		},
		{styleBtn(btnSeekBack, "seek_back", ColourBlue), styleBtn(btnSeekFwd, "seek_fwd", ColourGreen)},
		{styleBtn(autoplayBtnText(autoplayOn), "autoplay_toggle", ColourBlue)},
		{styleBtn(btnClose, "close", ColourRed)},
	}
}

func GetQueuedMarkup(chatID int64, index int) [][]InlineBtn {
	return [][]InlineBtn{
		{
			styleBtn(btnPlayNow, fmt.Sprintf("queue_now:%d:%d", chatID, index), ColourBlue),
			styleBtn(btnSkip, fmt.Sprintf("skip:%d", chatID), ColourGreen),
		},
		{styleBtn(btnClose, "close", ColourRed)},
	}
}
