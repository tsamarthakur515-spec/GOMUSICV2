package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/amarnathcjd/gogram/telegram"
)

var (
	noThumb   = map[int64]bool{}
	noThumbMu sync.Mutex
)

const (
	creditAPI   = "MUSIC API BY: ARUYT API"
	creditBot   = "MUSIC BOT BY : PANDA BABY"
	creditOwner = "MUSIC API OWNER : PANDA BABY"
)

func thumbsOff(chatID int64) bool {
	noThumbMu.Lock()
	defer noThumbMu.Unlock()
	return noThumb[chatID]
}

func toggleThumbs(chatID int64) bool {
	noThumbMu.Lock()
	defer noThumbMu.Unlock()
	noThumb[chatID] = !noThumb[chatID]
	return noThumb[chatID]
}

func ffText(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `’`)
	s = strings.ReplaceAll(s, `:`, `\:`)
	s = strings.ReplaceAll(s, `%`, `%%`)
	return s
}

func pickFont(bold bool) string {
	cands := []string{
		"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
		"/usr/share/fonts/truetype/freefont/FreeSansBold.ttf",
		"/usr/share/fonts/truetype/freefont/FreeSans.ttf",
	}
	if !bold {
		cands = []string{
			"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
			"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
			"/usr/share/fonts/truetype/freefont/FreeSans.ttf",
			"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
		}
	}
	for _, p := range cands {
		if st, err := os.Stat(p); err == nil && st.Size() > 0 {
			return p
		}
	}
	return ""
}

func dt(text, font string, size, x, y int, color string) string {
	part := "drawtext=text='" + ffText(text) + "':fontcolor=" + color +
		":fontsize=" + itoa(size) + ":x=" + itoa(x) + ":y=" + itoa(y) +
		":shadowcolor=black@0.55:shadowx=2:shadowy=2"
	if font != "" {
		part += ":fontfile=" + font
	}
	return part
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func makeEditedThumb(cover, title, duration, requester, videoID string) string {
	if cover == "" {
		return ""
	}
	_ = os.MkdirAll(downloadDir, 0o755)
	name := "edit.jpg"
	if len(videoID) >= 6 {
		name = "edit_" + videoID + ".jpg"
	}
	out := filepath.Join(downloadDir, name)

	title = shortTitle(strings.TrimSpace(title), 36)
	if title == "" {
		title = "Now Playing"
	}
	artist := shortTitle(strings.TrimSpace(requester), 28)
	if artist == "" {
		artist = BotName
	}
	if duration == "" {
		duration = "0:00"
	}
	label := strings.TrimSpace(BotName)
	if label == "" {
		label = "Music Bot"
	}
	bold := pickFont(true)
	reg := pickFont(false)

	fc := "[0:v]scale=1280:720:force_original_aspect_ratio=increase,crop=1280:720,gblur=sigma=26,eq=brightness=-0.18:saturation=0.9[bg];" +
		"[0:v]scale=288:288:force_original_aspect_ratio=increase,crop=288:288[art];" +
		"[bg]drawbox=x=80:y=100:w=1120:h=430:color=black@0.55:t=fill[card];" +
		"[card][art]overlay=120:160[v];" +
		"[v]drawbox=x=440:y=392:w=700:h=8:color=white@0.22:t=fill," +
		"drawbox=x=440:y=392:w=250:h=8:color=white@0.95:t=fill," +
		dt(label, reg, 22, 440, 175, "white@0.75") + "," +
		dt(title, bold, 38, 440, 215, "white") + "," +
		dt(artist, reg, 26, 440, 270, "white@0.85") + "," +
		dt("0:00", reg, 20, 440, 360, "white@0.8") + "," +
		dt(duration, reg, 20, 1040, 360, "white@0.8") + "," +
		dt(creditAPI, reg, 22, 80, 560, "white@0.92") + "," +
		dt(creditBot, reg, 22, 80, 598, "white@0.92") + "," +
		dt(creditOwner, reg, 22, 80, 636, "white@0.92")

	cmd := exec.Command("ffmpeg", "-y", "-i", cover,
		"-filter_complex", fc,
		"-frames:v", "1", "-q:v", "3", out)
	if err := cmd.Run(); err != nil {
		simple := "scale=1280:720:force_original_aspect_ratio=increase,crop=1280:720," +
			"eq=brightness=-0.12," +
			"drawbox=x=0:y=500:w=1280:h=220:color=black@0.6:t=fill," +
			dt(title, bold, 40, 48, 520, "white") + "," +
			dt(creditAPI, reg, 22, 48, 575, "white@0.92") + "," +
			dt(creditBot, reg, 22, 48, 610, "white@0.92") + "," +
			dt(creditOwner, reg, 22, 48, 645, "white@0.92")
		cmd = exec.Command("ffmpeg", "-y", "-i", cover, "-vf", simple, "-frames:v", "1", "-q:v", "3", out)
		if err2 := cmd.Run(); err2 != nil {
			cmd = exec.Command("ffmpeg", "-y", "-i", cover,
				"-vf", "scale=1280:720:force_original_aspect_ratio=increase,crop=1280:720",
				"-frames:v", "1", "-q:v", "3", out)
			if err3 := cmd.Run(); err3 != nil {
				return cover
			}
		}
	}
	return out
}

func handleNoThumb(m *telegram.NewMessage) error {
	if blocked(m) || m.IsPrivate() || !isAuthorized(m) {
		return nil
	}
	off := toggleThumbs(m.ChatID())
	state := smallcaps("hidden")
	if !off {
		state = smallcaps("shown")
	}
	_, _ = sendHTML(Bot, m.ChatID(), wrapBQ(smallcaps("thumbnails")+" : "+state), nil)
	return nil
}
