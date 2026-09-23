package main

import (
	"fmt"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/nikhil390u8o/GOMUSICV2/ntgcalls"
)

func RoomPlay(chatID int64, song Song, force bool, msg *telegram.NewMessage) error {
	unlock := lockChatPlay(chatID)
	defer unlock()

	if !force && isBusy(chatID) {
		pos := addToQueue(chatID, song)
		body := incomingTrackHTML(pos-1, song)
		if msg != nil {
			_ = editHTML(msg, body, gogramMarkup(GetQueuedMarkup(chatID, pos-1)))
		} else {
			_, _ = sendHTML(Bot, chatID, body, gogramMarkup(GetQueuedMarkup(chatID, pos-1)))
		}
		return nil
	}

	if peekCurrent(chatID) == nil || peekCurrent(chatID).URL != song.URL {
		addToQueue(chatID, song)
	}
	return playSongOpt(chatID, msg, song, force || shouldStayInCall(chatID) || isLiveSession(chatID))
}

func RoomNextTrack(chatID int64) *Song {
	done := popCurrent(chatID)
	if done != nil && done.FilePath != "" {
		go deleteFile(done.FilePath)
	}
	return peekCurrent(chatID)
}

func RoomChangeStream(chatID int64) error {
	holdSwitch(chatID)
	nxt := RoomNextTrack(chatID)
	if nxt == nil {
		releaseSwitch(chatID)
		leaveVCNow(chatID)
		_, _ = sendHTML(Bot, chatID, wrapBQ(smallcaps("the queue has finished")+"\n\n"+smallcaps("use /play to add more songs")), nil)
		return fmt.Errorf("queue empty")
	}
	msg, _ := sendHTML(Bot, chatID, wrapBQ(smallcaps("processing...")), nil)
	err := playSongOpt(chatID, msg, *nxt, true)
	releaseSwitch(chatID)
	return err
}

func RoomPlayNow(chatID int64, index int) error {
	holdSwitch(chatID)
	q := getQueue(chatID)
	if len(q) == 0 {
		releaseSwitch(chatID)
		return fmt.Errorf("queue empty")
	}
	if index <= 0 || index >= len(q) {
		if len(q) > 1 {
			index = 1
		} else {
			releaseSwitch(chatID)
			return fmt.Errorf("song not in queue")
		}
	}
	song := q[index]
	old := q[0]
	rest := make([]Song, 0, len(q)-1)
	for i, s := range q {
		if i == 0 || i == index {
			continue
		}
		rest = append(rest, s)
	}
	queueMu.Lock()
	chatQueues[chatID] = append([]Song{song}, rest...)
	queueMu.Unlock()
	if old.FilePath != "" && old.URL != song.URL {
		go deleteFile(old.FilePath)
	}
	msg, _ := sendHTML(Bot, chatID, wrapBQ(smallcaps("processing...")), nil)
	err := playSongOpt(chatID, msg, song, true)
	releaseSwitch(chatID)
	return err
}

func ntgPlaySameCall(chatID int64, media ntgcalls.MediaDescription) error {
	if Calls == nil {
		return fmt.Errorf("ntgcalls not ready")
	}
	if hasLocalCall(chatID) {
		return Calls.SetStreamSources(chatID, ntgcalls.CaptureStream, media)
	}
	return fmt.Errorf("no local call")
}
