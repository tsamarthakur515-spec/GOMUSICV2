package main

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

var (
	callIsVideo  = map[int64]bool{}
	connectWait  = map[int64]chan error{}
	connectMu    sync.Mutex
	currentPaths = map[int64]string{}
	streamGen    = map[int64]int{}
	streamAt     = map[int64]time.Time{}
)

func bumpStream(chatID int64) int {
	connectMu.Lock()
	defer connectMu.Unlock()
	streamGen[chatID]++
	streamAt[chatID] = time.Now()
	return streamGen[chatID]
}

func setCurrentPath(chatID int64, path string) {
	connectMu.Lock()
	currentPaths[chatID] = path
	connectMu.Unlock()
	queueMu.Lock()
	q := chatQueues[chatID]
	if len(q) > 0 {
		q[0].FilePath = path
		chatQueues[chatID] = q
	}
	queueMu.Unlock()
}

func currentPath(chatID int64) string {
	connectMu.Lock()
	defer connectMu.Unlock()
	return currentPaths[chatID]
}

func hasLocalCall(chatID int64) bool {
	return Calls != nil && Calls.Calls()[chatID] != nil
}

func leaveVC(chatID int64) {
	bumpStream(chatID)
	stopAutoplay(chatID)
	for _, song := range clearQueue(chatID) {
		deleteFile(song.FilePath)
	}
	if Calls != nil {
		_ = Calls.Stop(chatID)
	}
	if call, ok := activeCalls[chatID]; ok {
		_, _ = Assistant.PhoneLeaveGroupCall(call, 0)
		delete(activeCalls, chatID)
	}
	delete(callIsVideo, chatID)
}

func handleStreamEnd(chatID int64) {
	connectMu.Lock()
	started := streamAt[chatID]
	gen := streamGen[chatID]
	connectMu.Unlock()
	if started.IsZero() || time.Since(started) < 4*time.Second {
		return
	}
	time.Sleep(500 * time.Millisecond)
	connectMu.Lock()
	if streamGen[chatID] != gen {
		connectMu.Unlock()
		return
	}
	connectMu.Unlock()

	done := popCurrent(chatID)
	if done != nil {
		time.Sleep(time.Second)
		deleteFile(done.FilePath)
	}
	time.Sleep(time.Second)
	nxt := peekCurrent(chatID)
	if nxt != nil {
		msg, _ := sendHTML(Bot, chatID, wrapBQ(smallcaps("next track")+"\n"+richEsc(nxt.Title)), nil)
		_ = playSong(chatID, msg, *nxt)
		return
	}
	leaveVC(chatID)
	_, _ = sendHTML(Bot, chatID, wrapBQ(smallcaps("queue finished")), nil)
}

func ensureVC(chatID int64) error {
	if _, err := resolveActiveCall(chatID); err == nil {
		return nil
	}
	peer, err := Assistant.ResolvePeer(chatID)
	if err != nil {
		return err
	}
	_, err = Assistant.PhoneCreateGroupCall(&telegram.PhoneCreateGroupCallParams{
		Peer:     peer,
		RandomID: int32(rand.Intn(90000) + 10000),
	})
	if err != nil {
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "already") || strings.Contains(low, "groupcall_already_started") {
			return nil
		}
		return err
	}
	time.Sleep(2 * time.Second)
	return nil
}

func resolveActiveCall(chatID int64) (telegram.InputGroupCall, error) {
	if call, ok := activeCalls[chatID]; ok && call != nil {
		return call, nil
	}
	peer, err := Assistant.ResolvePeer(chatID)
	if err != nil {
		return nil, err
	}
	switch p := peer.(type) {
	case *telegram.InputPeerChannel:
		full, err := Assistant.ChannelsGetFullChannel(&telegram.InputChannelObj{ChannelID: p.ChannelID, AccessHash: p.AccessHash})
		if err != nil {
			return nil, err
		}
		cf, ok := full.FullChat.(*telegram.ChannelFull)
		if !ok || cf.Call == nil {
			return nil, errors.New("no active group call")
		}
		return cf.Call, nil
	case *telegram.InputPeerChat:
		full, err := Assistant.MessagesGetFullChat(p.ChatID)
		if err != nil {
			return nil, err
		}
		cf, ok := full.FullChat.(*telegram.ChatFullObj)
		if !ok || cf.Call == nil {
			return nil, errors.New("no active group call")
		}
		return cf.Call, nil
	default:
		return nil, fmt.Errorf("unsupported peer type %T", peer)
	}
}

func shouldRetryJoin(err error) bool {
	if err == nil {
		return false
	}
	low := strings.ToLower(err.Error())
	return strings.Contains(low, "interdc") ||
		strings.Contains(low, "timed out") ||
		strings.Contains(low, "timeout") ||
		strings.Contains(low, "flood") ||
		strings.Contains(low, "500") ||
		strings.Contains(low, "503")
}

func alreadyJoinedError(err error) bool {
	if err == nil {
		return false
	}
	low := strings.ToLower(err.Error())
	return strings.Contains(low, "already") ||
		strings.Contains(low, "participant_join") ||
		strings.Contains(low, "joined")
}

func startNTGStream(chatID int64, path string, video bool, seekSec int) error {
	if Calls == nil {
		return errors.New("ntgcalls not ready")
	}
	media := buildMedia(path, video, seekSec)
	return startNTGStreamWithMedia(chatID, media, video)
}
