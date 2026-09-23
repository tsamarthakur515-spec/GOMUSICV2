package main

import (
	"log"
	"sync"
)

var (
	apActive   = map[int64]bool{}
	apQuery    = map[int64]string{}
	apFetched  = map[int64]map[string]bool{}
	apFetching = map[int64]bool{}
	apMu       sync.Mutex
)

const (
	batchSize        = 10
	refetchThreshold = 2
)

func isAutoplay(chatID int64) bool {
	apMu.Lock()
	defer apMu.Unlock()
	return apActive[chatID]
}

func getAutoplayQuery(chatID int64) string {
	apMu.Lock()
	defer apMu.Unlock()
	return apQuery[chatID]
}

func autoplayFetching(chatID int64) bool {
	apMu.Lock()
	defer apMu.Unlock()
	return apFetching[chatID]
}

func stopAutoplay(chatID int64) {
	apMu.Lock()
	defer apMu.Unlock()
	delete(apActive, chatID)
	delete(apQuery, chatID)
	delete(apFetched, chatID)
	delete(apFetching, chatID)
}

func startAutoplay(chatID int64, query, requester string, requesterID int64) int {
	stopAutoplay(chatID)
	apMu.Lock()
	apActive[chatID] = true
	apQuery[chatID] = query
	apFetched[chatID] = map[string]bool{}
	apMu.Unlock()
	return fetchMore(chatID, requester, requesterID)
}

func toggleAutoplay(chatID int64) bool {
	if isAutoplay(chatID) {
		stopAutoplay(chatID)
		return false
	}
	cur := peekCurrent(chatID)
	query := "trending songs"
	req := "AutoPlay"
	var reqID int64
	if cur != nil {
		if cur.Title != "" {
			query = cur.Title
		}
		if cur.Requester != "" {
			req = cur.Requester
		}
		reqID = cur.RequesterID
	}
	startAutoplay(chatID, query, req, reqID)
	return true
}

func fetchMore(chatID int64, requester string, requesterID int64) int {
	apMu.Lock()
	if apFetching[chatID] {
		apMu.Unlock()
		return 0
	}
	apFetching[chatID] = true
	query := apQuery[chatID]
	fetched := apFetched[chatID]
	if fetched == nil {
		fetched = map[string]bool{}
		apFetched[chatID] = fetched
	}
	apMu.Unlock()
	defer func() {
		apMu.Lock()
		apFetching[chatID] = false
		apMu.Unlock()
	}()

	variations := []string{query, query + " new songs", query + " best songs", query + " hits", query + " latest"}
	added := 0
	for _, variation := range variations {
		if !isAutoplay(chatID) || added >= batchSize {
			break
		}
		urlStr, title, durISO, thumb, playlist, err := searchYT(variation)
		if err != nil {
			log.Println("[AutoPlay] fetch variation", variation, err)
			continue
		}
		items := playlist
		if len(items) == 0 && urlStr != "" {
			items = []ytItem{{Link: urlStr, Title: title, Duration: durISO, Thumbnail: thumb}}
		}
		for _, item := range items {
			if !isAutoplay(chatID) || added >= batchSize {
				break
			}
			vid := extractVideoID(item.Link)
			apMu.Lock()
			if fetched[vid] {
				apMu.Unlock()
				continue
			}
			fetched[vid] = true
			apMu.Unlock()
			addToQueue(chatID, Song{
				URL:             item.Link,
				Title:           item.Title,
				Duration:        isoToHuman(item.Duration),
				DurationSeconds: isoToSec(item.Duration),
				Requester:       "AutoPlay",
				RequesterID:     requesterID,
				Thumbnail:       item.Thumbnail,
			})
			added++
		}
	}
	log.Printf("[AutoPlay] %d: +%d songs added\n", chatID, added)
	return added
}

func maybeRefetch(chatID int64, requester string, requesterID int64) {
	if !isAutoplay(chatID) || autoplayFetching(chatID) {
		return
	}
	if queueSize(chatID) <= refetchThreshold {
		go fetchMore(chatID, requester, requesterID)
	}
}
