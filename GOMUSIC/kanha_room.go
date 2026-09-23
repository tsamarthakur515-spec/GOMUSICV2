package main

import (
	"fmt"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/nikhil390u8o/GOMUSICV2/ntgcalls"
)

// RoomPlay mirrors Kanha RoomState.Play:
//   - if something is already playing and force is false, only append to queue
//   - otherwise start/replace the stream on the SAME ntgcalls call (no leave/rejoin)
func RoomPlay(chatID int64, song Song, force bool, msg *telegram.NewMessage) error {
	if !force && isBusy(chatID) {
		pos := addToQueue(chatID, song)
		body := smallcaps("added to queue") + "\n\n" +
			smallcaps("title") + " : " + richEsc(shortTitle(song.Title, 42)) + "\n" +
			smallcaps("duration") + " : " + richEsc(song.Duration) + "\n" +
			smallcaps("position") + " : " + fmt.Sprintf("%d", pos)
		if msg != nil {
			_ = editHTML(msg, wrapBQ(body), gogramMarkup(GetQueuedMarkup(chatID, pos-1)))
		} else {
			_, _ = sendHTML(Bot, chatID, wrapBQ(body), gogramMarkup(GetQueuedMarkup(chatID, pos-1)))
		}
		return nil
	}

	if peekCurrent(chatID) == nil || peekCurrent(chatID).URL != song.URL {
		addToQueue(chatID, song)
	}
	return playSongOpt(chatID, msg, song, force || shouldStayInCall(chatID))
}

// RoomNextTrack mirrors Kanha NextTrack: drop current, return next queued song.
func RoomNextTrack(chatID int64) *Song {
	done := popCurrent(chatID)
	if done != nil {
		deleteFile(done.FilePath)
	}
	return peekCurrent(chatID)
}

// RoomChangeStream mirrors Kanha skip: next track + force Play on the same call.
func RoomChangeStream(chatID int64) error {
	beginSwitch(chatID)
	nxt := RoomNextTrack(chatID)
	if nxt == nil {
		leaveVCNow(chatID)
		_, _ = sendHTML(Bot, chatID, wrapBQ(smallcaps("queue is empty, left vc")), nil)
		return fmt.Errorf("queue empty")
	}
	msg, _ := sendHTML(Bot, chatID, wrapBQ(smallcaps("processing...")), nil)
	return playSongOpt(chatID, msg, *nxt, true)
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
