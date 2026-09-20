package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/nikhil390u8o/GOMUSICV2/ntgcalls"
)

var (
	callIsVideo  = map[int64]bool{}
	connectWait  = map[int64]chan error{}
	connectMu    sync.Mutex
	currentPaths = map[int64]string{}
)

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

func leaveVC(chatID int64) {
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
	done := popCurrent(chatID)
	if done != nil {
		time.Sleep(time.Second)
		deleteFile(done.FilePath)
	}
	time.Sleep(2 * time.Second)
	nxt := peekCurrent(chatID)
	if nxt != nil {
		msg, _ := sendHTML(Bot, chatID, richHeading("next track", 3)+richNote(richEsc(nxt.Title)), nil)
		_ = playSong(chatID, msg, *nxt)
		return
	}
	leaveVC(chatID)
	_, _ = sendHTML(Bot, chatID, richHeading("queue finished", 3), nil)
}

func ensureVC(chatID int64) error {
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

func startNTGStream(chatID int64, path string, video bool, seekSec int) error {
	if Calls == nil {
		return errors.New("ntgcalls not ready")
	}
	media := buildMedia(path, video, seekSec)
	if Calls.Calls()[chatID] != nil {
		return Calls.SetStreamSources(chatID, ntgcalls.CaptureStream, media)
	}

	var last error
	for attempt := 1; attempt <= 4; attempt++ {
		inputCall, err := resolveActiveCall(chatID)
		if err != nil {
			if e := ensureVC(chatID); e == nil {
				inputCall, err = resolveActiveCall(chatID)
			}
			if err != nil {
				last = err
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
		}

		local, err := Calls.CreateCall(chatID)
		if err != nil {
			_ = Calls.Stop(chatID)
			last = err
			time.Sleep(2 * time.Second)
			continue
		}
		if err := Calls.SetStreamSources(chatID, ntgcalls.CaptureStream, media); err != nil {
			_ = Calls.Stop(chatID)
			last = err
			time.Sleep(2 * time.Second)
			continue
		}

		me, err := Assistant.GetMe()
		if err != nil {
			_ = Calls.Stop(chatID)
			return err
		}
		wait := make(chan error, 1)
		connectMu.Lock()
		connectWait[chatID] = wait
		connectMu.Unlock()

		updates, err := Assistant.PhoneJoinGroupCall(&telegram.PhoneJoinGroupCallParams{
			Call:         inputCall,
			JoinAs:       &telegram.InputPeerUser{UserID: me.ID, AccessHash: me.AccessHash},
			VideoStopped: !video,
			Muted:        false,
			Params:       &telegram.DataJson{Data: local},
		})
		if err != nil {
			_ = Calls.Stop(chatID)
			last = err
			log.Printf("PhoneJoinGroupCall try %d/4: %v", attempt, err)
			if shouldRetryJoin(err) && attempt < 4 {
				time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
				continue
			}
			return err
		}
		remote := "{\"transport\": null}"
		if u, ok := updates.(*telegram.UpdatesObj); ok {
			for _, upd := range u.Updates {
				if conn, ok := upd.(*telegram.UpdateGroupCallConnection); ok && conn.Params != nil {
					remote = conn.Params.Data
				}
			}
		}
		if err := Calls.Connect(chatID, remote, false); err != nil {
			_ = Calls.Stop(chatID)
			last = err
			time.Sleep(2 * time.Second)
			continue
		}
		select {
		case err := <-wait:
			if err != nil {
				last = err
				_ = Calls.Stop(chatID)
				time.Sleep(2 * time.Second)
				continue
			}
		case <-time.After(20 * time.Second):
		}
		activeCalls[chatID] = inputCall
		callIsVideo[chatID] = video
		return nil
	}
	if last == nil {
		last = errors.New("join call failed")
	}
	return last
}
