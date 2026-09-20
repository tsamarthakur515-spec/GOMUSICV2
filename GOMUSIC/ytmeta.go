package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func init() {
	ytIDRe = regexp.MustCompile(`(?:v=|youtu\.be/|/vi/)([A-Za-z0-9_-]{11})`)
}

func probeDuration(path string) float64 {
	out, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", path).Output()
	if err != nil {
		return 0
	}
	n, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	return n
}

func formatClock(secs float64) string {
	if secs < 0 {
		secs = 0
	}
	t := int(secs + 0.5)
	h := t / 3600
	m := (t % 3600) / 60
	s := t % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func oembedTitle(id string) string {
	if len(id) != 11 {
		return ""
	}
	u := "https://www.youtube.com/oembed?format=json&url=https://www.youtube.com/watch?v=" + id
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ""
	}
	var data struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ""
	}
	return strings.TrimSpace(data.Title)
}

func searchSmart(query string) (string, string, string, string, []ytItem, error) {
	q := strings.TrimSpace(query)
	if item, err := searchViaYTDLP(q); err == nil && item.Link != "" {
		return item.Link, item.Title, item.Duration, item.Thumbnail, nil, nil
	}
	return searchYT(q)
}

func searchViaYTDLP(query string) (ytItem, error) {
	args := []string{
		"ytsearch1:" + query,
		"--no-warnings",
		"--skip-download",
		"--print", "%(id)s\t%(title)s\t%(duration_string)s",
	}
	if st, err := os.Stat("cookies.txt"); err == nil && st.Size() > 10 {
		args = append([]string{"--cookies", "cookies.txt"}, args...)
	}
	out, err := exec.Command("yt-dlp", args...).CombinedOutput()
	if err != nil {
		return ytItem{}, fmt.Errorf("ytsearch: %v: %s", err, strings.TrimSpace(string(out)))
	}
	line := strings.TrimSpace(string(out))
	if i := strings.LastIndex(line, "\n"); i >= 0 {
		line = strings.TrimSpace(line[i+1:])
	}
	parts := strings.Split(line, "\t")
	if len(parts) < 1 || len(parts[0]) != 11 {
		return ytItem{}, fmt.Errorf("ytsearch no id: %s", line)
	}
	id := parts[0]
	title := "YouTube Video"
	dur := "0:00"
	if len(parts) > 1 && parts[1] != "" && parts[1] != "NA" {
		title = parts[1]
	}
	if len(parts) > 2 && parts[2] != "" && parts[2] != "NA" {
		dur = parts[2]
	}
	link := "https://www.youtube.com/watch?v=" + id
	return ytItem{Link: link, Title: title, Duration: dur, Thumbnail: thumbFor(link, "")}, nil
}
