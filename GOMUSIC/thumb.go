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
		case 'ꜱ':
			b.WriteByte('S')
		case 'ᴛ':
			b.WriteByte('T')
		case 'ᴜ':
			b.WriteByte('U')
		case 'ᴠ':
			b.WriteByte('V')
		case 'ᴡ':
			b.WriteByte('W')
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
	s = strings.ReplaceAll(s, "\u00d7", "x")
	s = strings.ReplaceAll(s, "\u2022", "-")
	s = strings.ReplaceAll(s, "\u2014", "-")
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
		":shadowcolor=black@0.35:shadowx=1:shadowy=1"
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

func roundAlpha(radius, alpha int) string {
	r := itoa(radius)
	a := itoa(alpha)
	return "if(gt(hypot(X-min(max(X," + r + "),W-1-" + r + "),Y-min(max(Y," + r + "),H-1-" + r + "))," + r + "),0," + a + ")"
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

	title = shortTitle(safeDrawText(title), 34)
	if title == "" {
		title = "Now Playing"
	}
	who := shortTitle(safeDrawText(requester), 28)
	if who == "" {
		who = safeDrawText(BotName)
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

	cx, cy, cw, ch := 160, 118, 960, 484
	cardR := 48
	art := 200
	artR := 28
	ax, ay := cx+40, cy+40
	tx := ax + art + 32
	progressY := cy + 268
	ctrlY := cy + 318
	volY := cy + 412
	barX, barW := tx, 620

	cardA := roundAlpha(cardR, 158)
	artA := roundAlpha(artR, 255)

	fc := "[0:v]scale=1280:720:force_original_aspect_ratio=increase,crop=1280:720," +
		"gblur=sigma=22,eq=brightness=-0.10:saturation=0.88[bg];" +
		"color=c=black:s=" + itoa(cw) + "x" + itoa(ch) + ":d=1:r=1[cardbase];" +
		"[cardbase]format=yuva444p,geq=lum=0:cb=128:cr=128:a='" + cardA + "'[card];" +
		"[bg][card]overlay=" + itoa(cx) + ":" + itoa(cy) + "[withcard];" +
		"[0:v]scale=" + itoa(art) + ":" + itoa(art) + ":force_original_aspect_ratio=increase,crop=" + itoa(art) + ":" + itoa(art) + "[artraw];" +
		"[artraw]format=yuva444p,geq=lum='p(X,Y)':cb='cb(X,Y)':cr='cr(X,Y)':a='" + artA + "'[art];" +
		"[withcard][art]overlay=" + itoa(ax) + ":" + itoa(ay) + "[v];" +
		"[v]" +
		"drawbox=x=" + itoa(barX) + ":y=" + itoa(progressY) + ":w=" + itoa(barW) + ":h=6:color=white@0.22:t=fill," +
		"drawbox=x=" + itoa(barX) + ":y=" + itoa(progressY) + ":w=" + itoa(barW*28/100) + ":h=6:color=white@0.95:t=fill," +
		"drawbox=x=" + itoa(cx+cw/2-28) + ":y=" + itoa(ctrlY) + ":w=14:h=34:color=white@0.96:t=fill," +
		"drawbox=x=" + itoa(cx+cw/2+6) + ":y=" + itoa(ctrlY) + ":w=14:h=34:color=white@0.96:t=fill," +
		"drawbox=x=" + itoa(ax) + ":y=" + itoa(volY) + ":w=" + itoa(cw-80) + ":h=5:color=white@0.18:t=fill," +
		"drawbox=x=" + itoa(ax) + ":y=" + itoa(volY) + ":w=" + itoa((cw-80)*42/100) + ":h=5:color=white@0.70:t=fill," +
		dt(label, reg, 16, tx, cy+46, "white@0.62") + "," +
		dt(title, bold, 30, tx, cy+78, "white") + "," +
		dt(who, reg, 20, tx, cy+128, "white@0.78") + "," +
		dt("0:00", reg, 15, barX, progressY-28, "white@0.70") + "," +
		dt("-"+duration, reg, 15, barX+barW-70, progressY-28, "white@0.70") + "," +
		dt("<<", bold, 26, cx+cw/2-118, ctrlY+2, "white@0.90") + "," +
		dt(">>", bold, 26, cx+cw/2+48, ctrlY+2, "white@0.90") + "," +
		dt(creditAPI, reg, 13, ax, cy+ch-36, "white@0.45")

	cmd := exec.Command("ffmpeg", "-y", "-i", cover,
		"-filter_complex", fc,
		"-frames:v", "1", "-q:v", "3", out)
	if err := cmd.Run(); err != nil {
		fc420 := strings.ReplaceAll(fc, "yuva444p", "yuva420p")
		cmd = exec.Command("ffmpeg", "-y", "-i", cover,
			"-filter_complex", fc420,
			"-frames:v", "1", "-q:v", "3", out)
		if err2 := cmd.Run(); err2 != nil {
			simple := "scale=1280:720:force_original_aspect_ratio=increase,crop=1280:720," +
				"gblur=sigma=20,eq=brightness=-0.12," +
				"drawbox=x=" + itoa(cx) + ":y=" + itoa(cy) + ":w=" + itoa(cw) + ":h=" + itoa(ch) + ":color=black@0.58:t=fill," +
				dt(label, reg, 16, tx, cy+46, "white@0.62") + "," +
				dt(title, bold, 30, tx, cy+78, "white") + "," +
				dt(who, reg, 20, tx, cy+128, "white@0.78")
			cmd = exec.Command("ffmpeg", "-y", "-i", cover, "-vf", simple, "-frames:v", "1", "-q:v", "3", out)
			if err3 := cmd.Run(); err3 != nil {
				cmd = exec.Command("ffmpeg", "-y", "-i", cover,
					"-vf", "scale=1280:720:force_original_aspect_ratio=increase,crop=1280:720",
					"-frames:v", "1", "-q:v", "3", out)
				if err4 := cmd.Run(); err4 != nil {
					return cover
				}
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
