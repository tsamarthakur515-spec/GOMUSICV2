package main

import "sync"

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

func startStore() {}

func addServedChat(chatID int64) {
	memMu.Lock()
	servedChats[chatID] = struct{}{}
	memMu.Unlock()
}

func addServedUser(userID int64) {
	memMu.Lock()
	servedUsers[userID] = struct{}{}
	memMu.Unlock()
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
