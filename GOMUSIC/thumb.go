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
	line1 := shortTitle(title, 42)
	if line1 == "" {
		line1 = "Now Playing"
	}
	line2 := strings.TrimSpace(duration)
	if requester != "" {
		if line2 != "" {
			line2 += "  •  " + shortTitle(requester, 18)
		} else {
			line2 = shortTitle(requester, 22)
		}
	}
	filter := "scale=1280:720:force_original_aspect_ratio=increase,crop=1280:720," +
		"eq=brightness=-0.08:saturation=1.08," +
		"drawbox=x=0:y=540:w=1280:h=180:color=black@0.55:t=fill," +
		"drawtext=text='" + ffText(line1) + "':fontcolor=white:fontsize=42:x=48:y=575:shadowcolor=black@0.6:shadowx=2:shadowy=2"
	if line2 != "" {
		filter += ",drawtext=text='" + ffText(line2) + "':fontcolor=white@0.9:fontsize=28:x=48:y=635:shadowcolor=black@0.5:shadowx=1:shadowy=1"
	}
	cmd := exec.Command("ffmpeg", "-y", "-i", cover,
		"-vf", filter,
		"-frames:v", "1", "-q:v", "3", out)
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("ffmpeg", "-y", "-i", cover,
			"-vf", "scale=1280:720:force_original_aspect_ratio=increase,crop=1280:720",
			"-frames:v", "1", "-q:v", "3", out)
		if err2 := cmd.Run(); err2 != nil {
			return cover
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
