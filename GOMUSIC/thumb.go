package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

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

func unfoldSmallcaps(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case 'ᴀ':
			b.WriteByte('A')
		case 'ʙ':
			b.WriteByte('B')
		case 'ᴄ':
			b.WriteByte('C')
		case 'ᴅ':
			b.WriteByte('D')
		case 'ᴇ':
			b.WriteByte('E')
		case 'ꜰ', 'ғ':
			b.WriteByte('F')
		case 'ɢ':
			b.WriteByte('G')
		case 'ʜ':
			b.WriteByte('H')
		case 'ɪ':
			b.WriteByte('I')
		case 'ᴊ':
			b.WriteByte('J')
		case 'ᴋ':
			b.WriteByte('K')
		case 'ʟ':
			b.WriteByte('L')
		case 'ᴍ':
			b.WriteByte('M')
		case 'ɴ':
			b.WriteByte('N')
		case 'ᴏ':
			b.WriteByte('O')
		case 'ᴘ':
			b.WriteByte('P')
		case 'ǫ', 'ϙ':
			b.WriteByte('Q')
		case 'ʀ':
			b.WriteByte('R')
		case 'ꜱ', '𝓼', '𝗌':
			b.WriteByte('S')
		case 'ᴛ':
			b.WriteByte('T')
		case 'ᴜ':
			b.WriteByte('U')
		case 'ᴠ':
			b.WriteByte('V')
		case 'ᴡ':
			b.WriteByte('W')
		case 'x', 'ᴇx':
			b.WriteRune(r)
		case 'ʏ':
			b.WriteByte('Y')
		case 'ᴢ':
			b.WriteByte('Z')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func safeDrawText(s string) string {
	s = unfoldSmallcaps(s)
	s = strings.ReplaceAll(s, "×", "x")
	s = strings.ReplaceAll(s, "•", "-")
	s = strings.ReplaceAll(s, "—", "-")
	s = strings.ReplaceAll(s, "|", "-")
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteByte(' ')
			continue
		}
		if r < 32 {
			continue
		}
		if r < 127 || unicode.Is(unicode.Latin, r) || unicode.IsNumber(r) || unicode.IsSpace(r) || strings.ContainsRune(".,!?'-_+()/&", r) {
			b.WriteRune(r)
			continue
		}
	}
	return strings.Join(strings.Fields(strings.TrimSpace(b.String())), " ")
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
		":shadowcolor=black@0.45:shadowx=1:shadowy=1"
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

	title = shortTitle(safeDrawText(title), 32)
	if title == "" {
		title = "Now Playing"
	}
	who := shortTitle(safeDrawText(requester), 22)
	artist := "Requested by " + who
	if who == "" {
		artist = safeDrawText(BotName)
	}
	if duration == "" {
		duration = "0:00"
	} else {
		duration = safeDrawText(duration)
	}
	label := safeDrawText(BotName)
	if label == "" {
		label = "Music Bot"
	}
	bold := pickFont(true)
	reg := pickFont(false)

	// Compact centered player card (closer to the phone-player screenshot).
	fc := "[0:v]scale=1280:720:force_original_aspect_ratio=increase,crop=1280:720,gblur=sigma=32,eq=brightness=-0.22:saturation=0.85[bg];" +
		"[0:v]scale=240:240:force_original_aspect_ratio=increase,crop=200:200[art];" +
		"[bg]drawbox=x=210:y=145:w=860:h=370:color=black@0.58:t=fill[card];" +
		"[card][art]overlay=250:195[v];" +
		"[v]drawbox=x=480:y=368:w=540:h=6:color=white@0.22:t=fill," +
		"drawbox=x=480:y=368:w=190:h=6:color=white@0.95:t=fill," +
		"drawbox=x=718:y=418:w=18:h=28:color=white@0.95:t=fill," +
		"drawbox=x=744:y=418:w=18:h=28:color=white@0.95:t=fill," +
		dt(label, reg, 18, 480, 198, "white@0.72") + "," +
		dt(title, bold, 30, 480, 232, "white") + "," +
		dt(artist, reg, 20, 480, 278, "white@0.82") + "," +
		dt("0:00", reg, 16, 480, 340, "white@0.8") + "," +
		dt(duration, reg, 16, 960, 340, "white@0.8") + "," +
		dt("<<", bold, 26, 560, 412, "white") + "," +
		dt(">>", bold, 26, 820, 412, "white") + "," +
		dt(creditAPI, reg, 20, 70, 555, "white@0.92") + "," +
		dt(creditBot, reg, 20, 70, 592, "white@0.92") + "," +
		dt(creditOwner, reg, 20, 70, 629, "white@0.92")

	cmd := exec.Command("ffmpeg", "-y", "-i", cover,
		"-filter_complex", fc,
		"-frames:v", "1", "-q:v", "3", out)
	if err := cmd.Run(); err != nil {
		simple := "scale=1280:720:force_original_aspect_ratio=increase,crop=1280:720," +
			"gblur=sigma=18,eq=brightness=-0.16," +
			"drawbox=x=200:y=160:w=880:h=320:color=black@0.55:t=fill," +
			dt(title, bold, 34, 240, 220, "white") + "," +
			dt(artist, reg, 22, 240, 280, "white@0.85") + "," +
			dt(creditAPI, reg, 20, 70, 555, "white@0.92") + "," +
			dt(creditBot, reg, 20, 70, 592, "white@0.92") + "," +
			dt(creditOwner, reg, 20, 70, 629, "white@0.92")
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
