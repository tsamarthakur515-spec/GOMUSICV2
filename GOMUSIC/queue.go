package main

import "sync"

type Song struct {
	URL             string
	Title           string
	Duration        string
	DurationSeconds int
	Requester       string
	RequesterID     int64
	Thumbnail       string
	Video           bool
	FilePath        string
}

var (
	chatQueues = map[int64][]Song{}
	queueMu    sync.Mutex
)

func getQueue(chatID int64) []Song {
	queueMu.Lock()
	defer queueMu.Unlock()
	q := chatQueues[chatID]
	out := make([]Song, len(q))
	copy(out, q)
	return out
}

func addToQueue(chatID int64, song Song) int {
	queueMu.Lock()
	defer queueMu.Unlock()
	chatQueues[chatID] = append(chatQueues[chatID], song)
	return len(chatQueues[chatID])
}

func popCurrent(chatID int64) *Song {
	queueMu.Lock()
	defer queueMu.Unlock()
	q := chatQueues[chatID]
	if len(q) == 0 {
		return nil
	}
	s := q[0]
	chatQueues[chatID] = q[1:]
	return &s
}

func removeFromQueue(chatID int64, index int) *Song {
	queueMu.Lock()
	defer queueMu.Unlock()
	q := chatQueues[chatID]
	if index < 0 || index >= len(q) {
		return nil
	}
	s := q[index]
	chatQueues[chatID] = append(q[:index], q[index+1:]...)
	return &s
}

func peekCurrent(chatID int64) *Song {
	queueMu.Lock()
	defer queueMu.Unlock()
	q := chatQueues[chatID]
	if len(q) == 0 {
		return nil
	}
	s := q[0]
	return &s
}

func peekNext(chatID int64) *Song {
	queueMu.Lock()
	defer queueMu.Unlock()
	q := chatQueues[chatID]
	if len(q) < 2 {
		return nil
	}
	s := q[1]
	return &s
}

func moveAfterCurrent(chatID int64, index int) bool {
	queueMu.Lock()
	defer queueMu.Unlock()
	q := chatQueues[chatID]
	if index <= 0 || index >= len(q) {
		return false
	}
	item := q[index]
	q = append(q[:index], q[index+1:]...)
	n := []Song{q[0], item}
	n = append(n, q[1:]...)
	chatQueues[chatID] = n
	return true
}

func clearQueue(chatID int64) []Song {
	queueMu.Lock()
	defer queueMu.Unlock()
	q := chatQueues[chatID]
	delete(chatQueues, chatID)
	return q
}

func queueSize(chatID int64) int {
	queueMu.Lock()
	defer queueMu.Unlock()
	return len(chatQueues[chatID])
}

func isEmpty(chatID int64) bool {
	return queueSize(chatID) == 0
}

func isBusy(chatID int64) bool {
	if queueSize(chatID) > 0 {
		return true
	}
	if Calls != nil {
		if _, ok := Calls.Calls()[chatID]; ok && Calls.Calls()[chatID] != nil {
			return true
		}
	}
	_, ok := activeCalls[chatID]
	return ok
}
