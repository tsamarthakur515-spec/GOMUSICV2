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
		case 'ꜱ', '𝓒', '𝘲':
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
