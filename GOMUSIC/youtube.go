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

func ytdlpBaseArgs() []string {
	args := []string{
		"--no-warnings", "--no-check-certificates", "--no-playlist",
		"--extractor-args", "youtube:player_client=android_music,mweb",
	}
	if st, err := os.Stat("cookies.txt"); err == nil && st.Size() > 10 {
		args = append(args, "--cookies", "cookies.txt")
	}
	return args
}

func resolveDirectURL(raw string, video bool) (streamSrc, error) {
	if st, err := os.Stat(raw); err == nil && !st.IsDir() && st.Size() > 1024 {
		return streamSrc{File: raw, Audio: raw, Video: raw}, nil
	}
	vid := extractVideoID(raw)
	link := raw
	if len(vid) == 11 {
		link = "https://www.youtube.com/watch?v=" + vid
	}
	format := "bestaudio[ext=m4a]/bestaudio/best"
	if video {
		format = "best[height<=480][ext=mp4]/best[height<=480]/best"
	}
	args := append(ytdlpBaseArgs(), "-f", format, "-g", "--max-downloads", "1", link)
	cmd := exec.Command("yt-dlp", args...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		lines := []string{}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
				lines = append(lines, line)
			}
		}
		if len(lines) == 1 {
			return streamSrc{Audio: lines[0], Video: lines[0]}, nil
		}
		if len(lines) >= 2 {
			return streamSrc{Video: lines[0], Audio: lines[1]}, nil
		}
	}
	path, derr := resolveStream(raw, video)
	if derr != nil {
		if err != nil {
			return streamSrc{}, fmt.Errorf("stream url: %v; download: %v", err, derr)
		}
		return streamSrc{}, derr
	}
	return streamSrc{File: path, Audio: path, Video: path}, nil
}

func resolveStream(raw string, video bool) (string, error) {
	if st, err := os.Stat(raw); err == nil && !st.IsDir() {
		return raw, nil
	}
	vid := extractVideoID(raw)
	_ = os.MkdirAll(downloadDir, 0o755)
	typ, ext := "audio", ".mp3"
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
	path, err := downloadViaAPI(vid, typ, fp)
	if err != nil {
		path, err = downloadViaYTDLP(vid, video, fp)
		if err != nil {
			return "", err
		}
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
	variants := []string{full, videoID}
	types := []string{typ}
	if typ == "video" {
		types = []string{"video", "mp4"}
	}
	client := &http.Client{Timeout: 15 * time.Minute}
	var last error
	for i := 0; i < 2; i++ {
		useURL := variants[i%len(variants)]
		useType := types[i%len(types)]
		u, _ := url.Parse(ShrutiAPIURL + "/download")
		q := u.Query()
		q.Set("url", useURL)
		q.Set("type", useType)
		q.Set("api_key", ShrutiAPIKey)
		u.RawQuery = q.Encode()
		resp, err := client.Get(u.String())
		if err != nil {
			last = err
			continue
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 400))
			resp.Body.Close()
			msg := strings.TrimSpace(string(body))
			last = fmt.Errorf("API HTTP %d: %s", resp.StatusCode, msg)
			if strings.Contains(strings.ToLower(msg), "416") || strings.Contains(strings.ToLower(msg), "range not satisfiable") {
				return "", last
			}
			continue
		}
		f, err := os.Create(dest)
		if err != nil {
			resp.Body.Close()
			return "", err
		}
		_, err = io.Copy(f, resp.Body)
		resp.Body.Close()
		f.Close()
		if err != nil {
			_ = os.Remove(dest)
			last = err
			continue
		}
		st, _ := os.Stat(dest)
		if st == nil || st.Size() < 1024 {
			_ = os.Remove(dest)
			last = fmt.Errorf("empty API file")
			continue
		}
		return dest, nil
	}
	if last == nil {
		last = fmt.Errorf("API download failed")
	}
	return "", last
}

func downloadViaYTDLP(videoID string, video bool, dest string) (string, error) {
	_ = os.Remove(dest)
	link := "https://www.youtube.com/watch?v=" + videoID
	outTmpl := dest
	if video {
		outTmpl = strings.TrimSuffix(dest, filepath.Ext(dest)) + ".%(ext)s"
	}
	args := append(ytdlpBaseArgs(), "-o", outTmpl, "--no-part")
	if video {
		args = append(args, "-f", "best[height<=480]/best")
	} else {
		args = append(args, "-f", "bestaudio/best", "-x", "--audio-format", "mp3")
	}
	args = append(args, link)
	cmd := exec.Command("yt-dlp", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("yt-dlp: %v: %s", err, strings.TrimSpace(string(out)))
	}
	if st, err := os.Stat(dest); err == nil && st.Size() > 1024 {
		return dest, nil
	}
	base := strings.TrimSuffix(dest, filepath.Ext(dest))
	for _, ext := range []string{".mp4", ".mkv", ".webm", ".mp3", ".m4a"} {
		p := base + ext
		if st, err := os.Stat(p); err == nil && st.Size() > 1024 {
			if p != dest {
				_ = os.Rename(p, dest)
				if _, err2 := os.Stat(dest); err2 == nil {
					return dest, nil
				}
				return p, nil
			}
			return p, nil
		}
	}
	return "", fmt.Errorf("yt-dlp finished but file missing")
}

func deleteFile(path string) {
	if path != "" && !isHTTP(path) {
		_ = os.Remove(path)
	}
}
