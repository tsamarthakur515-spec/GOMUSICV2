package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

var (
	memMu          sync.Mutex
	servedChats    = map[int64]struct{}{}
	servedUsers    = map[int64]struct{}{}
	playCounts     = map[int64]int64{}
	broadcastChats = map[int64]string{}
	blockedGroups  = map[int64]struct{}{}
	blockedUsers   = map[int64]struct{}{}
	chatEffects    = map[int64]effectState{}
)

const usersStoreFile = "data/users.json"

func startStore() {
	loadServedUsers()
}

func loadServedUsers() {
	b, err := os.ReadFile(usersStoreFile)
	if err != nil || len(b) == 0 {
		return
	}
	var ids []int64
	if json.Unmarshal(b, &ids) != nil {
		return
	}
	memMu.Lock()
	for _, id := range ids {
		if id != 0 {
			servedUsers[id] = struct{}{}
		}
	}
	memMu.Unlock()
}

func persistUsers() {
	memMu.Lock()
	ids := make([]int64, 0, len(servedUsers))
	for id := range servedUsers {
		ids = append(ids, id)
	}
	memMu.Unlock()
	_ = os.MkdirAll(filepath.Dir(usersStoreFile), 0o755)
	b, err := json.Marshal(ids)
	if err != nil {
		return
	}
	_ = os.WriteFile(usersStoreFile, b, 0o644)
}

func addServedChat(chatID int64) {
	memMu.Lock()
	servedChats[chatID] = struct{}{}
	memMu.Unlock()
}

func addServedUser(userID int64) bool {
	if userID == 0 {
		return false
	}
	memMu.Lock()
	_, exists := servedUsers[userID]
	servedUsers[userID] = struct{}{}
	memMu.Unlock()
	if !exists {
		persistUsers()
	}
	return !exists
}

func incrementPlayCount(chatID int64) {
	memMu.Lock()
	playCounts[chatID]++
	memMu.Unlock()
}

func getServedChatsCount() int64 {
	memMu.Lock()
	defer memMu.Unlock()
	return int64(len(servedChats))
}

func getServedUsersCount() int64 {
	memMu.Lock()
	defer memMu.Unlock()
	return int64(len(servedUsers))
}

func getTotalPlays() int64 {
	memMu.Lock()
	defer memMu.Unlock()
	var total int64
	for _, n := range playCounts {
		total += n
	}
	return total
}

func addBroadcastChat(chatID int64, kind string) {
	memMu.Lock()
	broadcastChats[chatID] = kind
	memMu.Unlock()
}

func getBroadcastChats() []int64 {
	memMu.Lock()
	defer memMu.Unlock()
	ids := make([]int64, 0, len(broadcastChats))
	for id := range broadcastChats {
		ids = append(ids, id)
	}
	return ids
}

func isGroupBlocked(chatID int64) bool {
	memMu.Lock()
	defer memMu.Unlock()
	_, ok := blockedGroups[chatID]
	return ok
}

func blockGroup(chatID int64) {
	memMu.Lock()
	blockedGroups[chatID] = struct{}{}
	memMu.Unlock()
}

func unblockGroup(chatID int64) {
	memMu.Lock()
	delete(blockedGroups, chatID)
	memMu.Unlock()
}

func getBlockedGroups() []int64 {
	memMu.Lock()
	defer memMu.Unlock()
	ids := make([]int64, 0, len(blockedGroups))
	for id := range blockedGroups {
		ids = append(ids, id)
	}
	return ids
}

func isUserBlockedDB(userID int64) bool {
	memMu.Lock()
	defer memMu.Unlock()
	_, ok := blockedUsers[userID]
	return ok
}

func blockUser(userID int64) {
	memMu.Lock()
	blockedUsers[userID] = struct{}{}
	memMu.Unlock()
}

func unblockUser(userID int64) {
	memMu.Lock()
	delete(blockedUsers, userID)
	memMu.Unlock()
}

func getBlockedUsers() []int64 {
	memMu.Lock()
	defer memMu.Unlock()
	ids := make([]int64, 0, len(blockedUsers))
	for id := range blockedUsers {
		ids = append(ids, id)
	}
	return ids
}

func saveChatEffects(chatID int64, speed float64, bass int, enabled bool) {
	memMu.Lock()
	chatEffects[chatID] = effectState{Speed: speed, Bass: bass, Enabled: enabled}
	memMu.Unlock()
}

func loadChatEffects(chatID int64) (speed float64, bass int, enabled bool) {
	memMu.Lock()
	defer memMu.Unlock()
	if s, ok := chatEffects[chatID]; ok {
		return s.Speed, s.Bass, s.Enabled
	}
	return 1.0, 0, false
}
