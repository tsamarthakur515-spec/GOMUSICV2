package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func fetchYouTubeMix(videoID string, limit int) []ytItem {
	videoID = extractVideoID(videoID)
	if len(videoID) != 11 {
		return nil
	}
	if limit <= 0 {
		limit = 15
	}
	body := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    "WEB",
				"clientVersion": "2.20240815.00.00",
				"hl":            "en",
			},
		},
		"playlistId": "RD" + videoID,
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, "https://www.youtube.com/youtubei/v1/next?prettyPrint=false", bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	var data any
	if err := json.Unmarshal(b, &data); err != nil {
		return nil
	}
	var out []ytItem
	walkMix(data, &out, limit)
	return out
}

func walkMix(v any, out *[]ytItem, limit int) {
	if len(*out) >= limit {
		return
	}
	switch t := v.(type) {
	case map[string]any:
		if vr, ok := t["playlistPanelVideoRenderer"].(map[string]any); ok {
			id, _ := vr["videoId"].(string)
			if len(id) == 11 {
				title := titleFromMap(vr)
				dur := durationFromMap(vr)
				link := "https://www.youtube.com/watch?v=" + id
				*out = append(*out, ytItem{Link: link, Title: title, Duration: dur, Thumbnail: thumbFor(link, "")})
			}
			return
		}
		for _, c := range t {
			walkMix(c, out, limit)
			if len(*out) >= limit {
				return
			}
		}
	case []any:
		for _, c := range t {
			walkMix(c, out, limit)
			if len(*out) >= limit {
				return
			}
		}
	}
}

func pickAutoplayCandidate(chatID int64, last *Song) *Song {
	if last == nil {
		return nil
	}
	vid := extractVideoID(last.URL)
	seen := map[string]bool{}
	if vid != "" {
		seen[vid] = true
	}
	add := func(item ytItem) *Song {
		id := extractVideoID(item.Link)
		if id == "" || seen[id] || apIsPlayed(chatID, id) {
			return nil
		}
		seen[id] = true
		apMarkPlayed(chatID, id)
		return &Song{
			URL:         item.Link,
			Title:       item.Title,
			Duration:    isoToHuman(item.Duration),
			Requester:   "🎵 ᴀᴜᴛᴏᴘʟᴀʏ",
			RequesterID: last.RequesterID,
			Thumbnail:   item.Thumbnail,
			Video:       last.Video,
		}
	}
	mix := fetchYouTubeMix(vid, 15)
	for i, item := range mix {
		if i >= 15 {
			break
		}
		if s := add(item); s != nil {
			return s
		}
	}
	query := strings.TrimSpace(last.Title)
	if query == "" {
		query = "trending songs"
	}
	for _, q := range []string{query, query + " songs", query + " mix"} {
		urlStr, title, dur, thumb, list, err := searchYT(q)
		if err != nil {
			continue
		}
		items := list
		if len(items) == 0 && urlStr != "" {
			items = []ytItem{{Link: urlStr, Title: title, Duration: dur, Thumbnail: thumb}}
		}
		for _, item := range items {
			if s := add(item); s != nil {
				return s
			}
		}
	}
	return nil
}

func takeAutoplayNext(chatID int64) *Song {
	if !isAutoplay(chatID) {
		return nil
	}
	cur := peekCurrent(chatID)
	if cur == nil {
		// last popped already; use stored query
		q := getAutoplayQuery(chatID)
		if q == "" {
			return nil
		}
		cur = &Song{Title: q, URL: q, Requester: "🎵 ᴀᴜᴛᴏᴘʟᴀʏ"}
	}
	s := pickAutoplayCandidate(chatID, cur)
	if s == nil {
		return nil
	}
	addToQueue(chatID, *s)
	log.Printf("[AutoPlay] queued mix track %s\n", s.Title)
	return peekCurrent(chatID)
}

func apMarkPlayed(chatID int64, videoID string) {
	if videoID == "" {
		return
	}
	apMu.Lock()
	defer apMu.Unlock()
	if apFetched[chatID] == nil {
		apFetched[chatID] = map[string]bool{}
	}
	apFetched[chatID][videoID] = true
}

func apIsPlayed(chatID int64, videoID string) bool {
	if videoID == "" {
		return false
	}
	apMu.Lock()
	defer apMu.Unlock()
	return apFetched[chatID][videoID]
}

func downloadWithRetry(raw string, video bool) (streamSrc, error) {
	var last error
	for i := 0; i < 3; i++ {
		src, err := resolveDirectURL(raw, video)
		if err == nil {
			if src.File != "" && !isHTTP(src.File) {
				st, e := os.Stat(src.File)
				if e != nil || st.Size() < 1024 {
					last = fmt.Errorf("invalid or empty file")
					continue
				}
			}
			return src, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("download failed")
	}
	return streamSrc{}, last
}
