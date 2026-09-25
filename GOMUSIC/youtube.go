package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const downloadDir = "downloads"

var (
	ytIDRe    = regexp.MustCompile(`(?:v=|youtu\.be/)([A-Za-z0-9_-]{11})`)
	htmlIDRe  = regexp.MustCompile(`"videoId":"([A-Za-z0-9_-]{11})"`)
	htmlTitle = regexp.MustCompile(`"title":\{"runs":\[\{"text":"([^"]+)"`)
)

type ytItem struct {
	Link, Title, Duration, Thumbnail string
}

type streamSrc struct {
	Audio string
	Video string
	File  string
}

func extractVideoID(raw string) string {
	raw = strings.TrimSpace(raw)
	if m := ytIDRe.FindStringSubmatch(raw); len(m) > 1 {
		return m[1]
	}
	if len(raw) == 11 && !strings.Contains(raw, " ") {
		return raw
	}
	return raw
}

func thumbFor(raw, existing string) string {
	existing = strings.TrimSpace(existing)
	if existing != "" && existing != "NA" && strings.HasPrefix(existing, "http") {
		return existing
	}
	id := extractVideoID(raw)
	if len(id) == 11 {
		return "https://i.ytimg.com/vi/" + id + "/hqdefault.jpg"
	}
	return ""
}

func cacheThumb(src string) string {
	if src == "" {
		return ""
	}
	if st, err := os.Stat(src); err == nil && !st.IsDir() && st.Size() > 0 {
		return src
	}
	_ = os.MkdirAll(downloadDir, 0o755)
	id := extractVideoID(src)
	name := "thumb.jpg"
	if len(id) == 11 {
		name = "t_" + id + ".jpg"
	}
	dest := filepath.Join(downloadDir, name)
	if st, err := os.Stat(dest); err == nil && st.Size() > 0 {
		return dest
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(src)
	if err != nil {
		return src
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return src
	}
	f, err := os.Create(dest)
	if err != nil {
		return src
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(dest)
		return src
	}
	return dest
}

func fileHasVideo(path string) bool {
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=codec_type", "-of", "csv=p=0", path).Output()
	return err == nil && strings.Contains(strings.ToLower(string(out)), "video")
}

func searchYT(query string) (urlStr, title, durISO, thumb string, playlist []ytItem, err error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "", "", "", "", nil, fmt.Errorf("empty query")
	}
	if strings.Contains(q, "youtube.com") || strings.Contains(q, "youtu.be") {
		id := extractVideoID(q)
		link := "https://www.youtube.com/watch?v=" + id
		return link, "YouTube Video", "0:00", thumbFor(link, ""), nil, nil
	}
	if item, e := searchInnertube(q); e == nil && item.Link != "" {
		return item.Link, item.Title, item.Duration, item.Thumbnail, nil, nil
	}
	if item, e := searchYouTubeHTML(q); e == nil && item.Link != "" {
		return item.Link, item.Title, item.Duration, item.Thumbnail, nil, nil
	}
	return "", "", "", "", nil, fmt.Errorf("search failed")
}

func searchInnertube(query string) (ytItem, error) {
	body := map[string]any{
		"context": map[string]any{"client": map[string]any{"clientName": "WEB", "clientVersion": "2.20240815.00.00", "hl": "en"}},
		"query":   query,
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, "https://www.youtube.com/youtubei/v1/search?prettyPrint=false", bytes.NewReader(raw))
	if err != nil {
		return ytItem{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ytItem{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ytItem{}, fmt.Errorf("innertube %d", resp.StatusCode)
	}
	var data any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ytItem{}, err
	}
	id, title, dur := walkYTSearch(data)
	if id == "" {
		return ytItem{}, fmt.Errorf("no video in innertube")
	}
	link := "https://www.youtube.com/watch?v=" + id
	return ytItem{Link: link, Title: title, Duration: dur, Thumbnail: thumbFor(link, "")}, nil
}

func walkYTSearch(v any) (id, title, dur string) {
	switch t := v.(type) {
	case map[string]any:
		if _, isVR := t["videoRenderer"]; isVR {
			vr := t["videoRenderer"].(map[string]any)
			if vid, ok := vr["videoId"].(string); ok && len(vid) == 11 {
				return vid, titleFromMap(vr), durationFromMap(vr)
			}
		}
		for _, c := range t {
			if id, title, dur = walkYTSearch(c); id != "" {
				return
			}
		}
	case []any:
		for _, c := range t {
			if id, title, dur = walkYTSearch(c); id != "" {
				return
			}
		}
	}
	return "", "", ""
}

func titleFromMap(m map[string]any) string {
	if t, ok := m["title"].(map[string]any); ok {
		if s, ok := t["simpleText"].(string); ok && s != "" {
			return s
		}
		if runs, ok := t["runs"].([]any); ok && len(runs) > 0 {
			if r, ok := runs[0].(map[string]any); ok {
				if s, ok := r["text"].(string); ok && s != "" {
					return s
				}
			}
		}
	}
	return "YouTube Video"
}

func durationFromMap(m map[string]any) string {
	if t, ok := m["lengthText"].(map[string]any); ok {
		if s, ok := t["simpleText"].(string); ok && s != "" {
			return s
		}
	}
	return "0:00"
}

func searchYouTubeHTML(query string) (ytItem, error) {
	u := "https://www.youtube.com/results?search_query=" + url.QueryEscape(query) + "&hl=en"
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ytItem{}, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	idm := htmlIDRe.FindSubmatch(b)
	if len(idm) < 2 {
		return ytItem{}, fmt.Errorf("no videoId in html")
	}
	id := string(idm[1])
	title := "YouTube Video"
	if tm := htmlTitle.FindSubmatch(b); len(tm) > 1 {
		title = string(tm[1])
	}
	link := "https://www.youtube.com/watch?v=" + id
	return ytItem{Link: link, Title: title, Duration: "0:00", Thumbnail: thumbFor(link, "")}, nil
}

func resolveDirectURL(raw string, video bool) (streamSrc, error) {
	if st, err := os.Stat(raw); err == nil && !st.IsDir() && st.Size() > 1024 {
		return streamSrc{File: raw, Audio: raw, Video: raw}, nil
	}
	path, err := resolveStream(raw, video)
	if err != nil {
		return streamSrc{}, err
	}
	return streamSrc{File: path, Audio: path, Video: path}, nil
}

func resolveStream(raw string, video bool) (string, error) {
	if st, err := os.Stat(raw); err == nil && !st.IsDir() {
		return raw, nil
	}
	vid := extractVideoID(raw)
	_ = os.MkdirAll(downloadDir, 0o755)
	typ, ext := "audio", ".m4a"
	if video {
		typ, ext = "video", ".mp4"
	}
	fp := filepath.Join(downloadDir, vid+ext)
	if st, err := os.Stat(fp); err == nil && st.Size() > 1024 {
		if !video || fileHasVideo(fp) {
			return fp, nil
		}
		_ = os.Remove(fp)
	}
	for _, old := range []string{".mp3", ".webm", ".mp4"} {
		p := filepath.Join(downloadDir, vid+old)
		if st, err := os.Stat(p); err == nil && st.Size() > 1024 {
			if !video || fileHasVideo(p) {
				return p, nil
			}
		}
	}
	path, err := downloadViaAPI(vid, typ, fp)
	if err != nil {
		return "", fmt.Errorf("download API: %v", err)
	}
	if video && !fileHasVideo(path) {
		_ = os.Remove(path)
		return "", fmt.Errorf("downloaded file has no video track")
	}
	return path, nil
}

func downloadViaAPI(videoID, typ, dest string) (string, error) {
	if ShrutiAPIURL == "" || ShrutiAPIKey == "" {
		return "", fmt.Errorf("download API not configured")
	}
	full := "https://www.youtube.com/watch?v=" + videoID
	client := &http.Client{Timeout: 15 * time.Minute}
	u, _ := url.Parse(ShrutiAPIURL + "/download")
	q := u.Query()
	q.Set("url", full)
	q.Set("type", typ)
	q.Set("api_key", ShrutiAPIKey)
	u.RawQuery = q.Encode()
	resp, err := client.Get(u.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 400))
		return "", fmt.Errorf("API HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		_ = os.Remove(dest)
		return "", err
	}
	st, _ := os.Stat(dest)
	if st == nil || st.Size() < 1024 {
		_ = os.Remove(dest)
		return "", fmt.Errorf("empty API file")
	}
	return dest, nil
}

func deleteFile(path string) {
	if path != "" && !isHTTP(path) {
		_ = os.Remove(path)
	}
}
