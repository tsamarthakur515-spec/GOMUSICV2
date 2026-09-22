package main

import (
	"fmt"
	"math/rand"
	"strings"
)

const (
	ColourRed   = "red"
	ColourBlue  = "blue"
	ColourGreen = "green"
)

var buttonColours = []string{ColourRed, ColourBlue, ColourGreen}

type InlineBtn struct {
	Text   string
	Data   string
	URL    string
	Colour string
}

func randomColour() string {
	return buttonColours[rand.Intn(len(buttonColours))]
}

func colourToStyle(colour string) string {
	switch strings.ToLower(colour) {
	case ColourRed:
		return "danger"
	case ColourGreen:
		return "success"
	default:
		return "primary"
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
	return styleBtn(text, cb, randomColour())
}

func UrlBtn(text, url string) InlineBtn {
	return styleURLBtn(text, url, randomColour())
}

func GetStartMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{UrlBtn("➕ "+smallcaps("add me in your group")+" ➕", BotLink+"?startgroup=true")},
		{UrlBtn(smallcaps("owner"), fmt.Sprintf("tg://user?id=%d", OwnerID)), DataBtn(smallcaps("about"), "about_menu")},
		{UrlBtn(smallcaps("support"), SupportGroup), UrlBtn(smallcaps("update"), UpdatesChannel)},
		{DataBtn(smallcaps("help and commands"), "show_help")},
	}
}

func GetAboutMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{DataBtn(smallcaps("back"), "go_back")},
	}
}

func GetHelpMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{DataBtn(smallcaps("admin"), "help_admin"), DataBtn(smallcaps("autoplay"), "help_autoplay"), DataBtn(smallcaps("gcast"), "help_gcast")},
		{DataBtn(smallcaps("bl-chat"), "help_blchat"), DataBtn(smallcaps("bl-users"), "help_blusers"), DataBtn(smallcaps("ping"), "help_ping")},
		{DataBtn(smallcaps("play"), "help_play"), DataBtn(smallcaps("speed"), "help_speed"), DataBtn(smallcaps("info"), "help_info")},
		{DataBtn(smallcaps("close"), "close_help")},
	}
}

func GetHelpHomeMarkup() [][]InlineBtn {
	rows := GetHelpMarkup()
	rows[len(rows)-1] = []InlineBtn{DataBtn(smallcaps("back"), "go_back")}
	return rows
}

func GetBackMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{DataBtn(smallcaps("back"), "show_help")},
		{DataBtn(smallcaps("close"), "close_help")},
	}
}

func GetGroupStartMarkup() [][]InlineBtn {
	return [][]InlineBtn{
		{DataBtn(smallcaps("help"), "show_help"), DataBtn(smallcaps("about"), "about_menu")},
	}
}
