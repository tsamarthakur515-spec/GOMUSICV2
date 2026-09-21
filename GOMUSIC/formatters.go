package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

var smallMap = map[rune]rune{
	'a': 'ᴀ', 'b': 'ʙ', 'c': 'ᴄ', 'd': 'ᴅ', 'e': 'ᴇ',
	'f': 'ꜰ', 'g': 'ɢ', 'h': 'ʜ', 'i': 'ɪ', 'j': 'ᴊ',
	'k': 'ᴋ', 'l': 'ʟ', 'm': 'ᴍ', 'n': 'ɴ', 'o': 'ᴏ',
	'p': 'ᴘ', 'q': 'ǫ', 'r': 'ʀ', 's': 's', 't': 'ᴛ',
	'u': 'ᴜ', 'v': 'ᴠ', 'w': 'ᴡ', 'x': 'x', 'y': 'ʏ', 'z': 'z',
}

func smallcaps(s string) string {
	var b strings.Builder
	b.Grow(len(s) * 2)
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			r = r - 'A' + 'a'
		}
		if m, ok := smallMap[r]; ok {
			b.WriteRune(m)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func wrapBQ(s string) string {
	// Photo caption edits drop expandable_blockquote. Normal blockquote stays.
	return "<blockquote>" + strings.TrimSpace(s) + "</blockquote>"
}

func fmtTime(seconds float64) string {
	s := int(seconds)
	m := s / 60
	h := m / 60
	s = s % 60
	m = m % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func parseDur(s string) int {
	if n := isoToSec(s); n > 0 {
		return n
	}
	if strings.Contains(s, ":") {
		parts := strings.Split(s, ":")
		nums := make([]int, 0, len(parts))
		for _, p := range parts {
			v, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil {
				return 0
			}
			nums = append(nums, v)
		}
		if len(nums) == 2 {
			return nums[0]*60 + nums[1]
		}
		if len(nums) == 3 {
			return nums[0]*3600 + nums[1]*60 + nums[2]
		}
	}
	return 0
}

func isoToSec(iso string) int {
	iso = strings.TrimSpace(iso)
	if !strings.HasPrefix(iso, "PT") {
		return 0
	}
	d, err := parseISO8601(iso)
	if err != nil {
		return 0
	}
	return int(d.Seconds())
}

func parseISO8601(iso string) (time.Duration, error) {
	s := strings.TrimPrefix(iso, "PT")
	var h, m, sec int
	num := ""
	for _, c := range s {
		if c >= '0' && c <= '9' {
			num += string(c)
			continue
		}
		v, _ := strconv.Atoi(num)
		num = ""
		switch c {
		case 'H':
			h = v
		case 'M':
			m = v
		case 'S':
			sec = v
		}
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second, nil
}

func isoToHuman(iso string) string {
	iso = strings.TrimSpace(iso)
	if iso == "" {
		return "?"
	}
	if strings.Contains(iso, ":") && !strings.HasPrefix(iso, "PT") {
		return iso
	}
	t := isoToSec(iso)
	if t == 0 {
		return "?"
	}
	return fmtTime(float64(t))
}

func secToISO(sec int) string {
	m, s := sec/60, sec%60
	h := m / 60
	m = m % 60
	if h > 0 {
		return fmt.Sprintf("PT%dH%dM%dS", h, m, s)
	}
	return fmt.Sprintf("PT%dM%dS", m, s)
}

func shortTitle(title string, n int) string {
	if n <= 0 {
		n = 22
	}
	r := []rune(title)
	if len(r) <= n {
		return title
	}
	return string(r[:n-1]) + "\u2026"
}

func progressBar(elapsed, total float64) string {
	if total <= 0 {
		return "N/A"
	}
	played := fmtTime(elapsed)
	dur := fmtTime(total)
	percentage := (elapsed / total) * 100
	umm := int(math.Floor(percentage))
	bar := "\u2014\u2014\u2014\u2014\u2014\u2014\u2014\u2014\u2014\u2661"
	switch {
	case umm <= 10:
		bar = "\u2661\u2014\u2014\u2014\u2014\u2014\u2014\u2014\u2014\u2014"
	case umm <= 20:
		bar = "\u2014\u2661\u2014\u2014\u2014\u2014\u2014\u2014\u2014\u2014"
	case umm <= 30:
		bar = "\u2014\u2014\u2661\u2014\u2014\u2014\u2014\u2014\u2014\u2014"
	case umm <= 40:
		bar = "\u2014\u2014\u2014\u2661\u2014\u2014\u2014\u2014\u2014\u2014"
	case umm <= 50:
		bar = "\u2014\u2014\u2014\u2014\u2661\u2014\u2014\u2014\u2014\u2014"
	case umm <= 60:
		bar = "\u2014\u2014\u2014\u2014\u2014\u2661\u2014\u2014\u2014\u2014"
	case umm <= 70:
		bar = "\u2014\u2014\u2014\u2014\u2014\u2014\u2661\u2014\u2014\u2014"
	case umm <= 80:
		bar = "\u2014\u2014\u2014\u2014\u2014\u2014\u2014\u2661\u2014\u2014"
	case umm <= 95:
		bar = "\u2014\u2014\u2014\u2014\u2014\u2014\u2014\u2014\u2661\u2014"
	}
	return played + " " + bar + " " + dur
}
