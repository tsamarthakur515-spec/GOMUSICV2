package main

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

func nowPlayingCaption(song Song) string {
	return streamNowPlayingHTML(song)
}

func nowPlayingKB(chatID int64, elapsed, total float64) telegram.ReplyMarkup {
	return gogramMarkup(GetNowPlayingMarkup(progressBar(elapsed, total), isAutoplay(chatID)))
}

func sendNowPlaying(chatID int64, song Song) *telegram.NewMessage {
	caption := nowPlayingCaption(song)
	kb := nowPlayingKB(chatID, 0, float64(parseDur(song.Duration)))
	if thumbsOff(chatID) {
		msg, _ := Bot.SendMessage(chatID, caption, &telegram.SendOptions{ParseMode: "HTML", ReplyMarkup: kb})
		return msg
	}
	vid := extractVideoID(song.URL)
	cover := cacheThumb(thumbFor(song.URL, song.Thumbnail))
	thumb := makeEditedThumb(cover, song.Title, song.Duration, song.Requester, vid)
	if thumb != "" {
		if abs, err := filepath.Abs(thumb); err == nil {
			thumb = abs
		}
		msg, err := Bot.SendMedia(chatID, thumb, &telegram.MediaOptions{
			Caption:     caption,
			ParseMode:   "HTML",
			ReplyMarkup: kb,
		})
		if err == nil {
			return msg
		}
		log.Println("panel upload failed:", err)
	}
	msg, _ := Bot.SendMessage(chatID, caption, &telegram.SendOptions{ParseMode: "HTML", ReplyMarkup: kb})
	return msg
}

func shouldStayInCall(chatID int64) bool {
	return hasLocalCall(chatID) && isLiveSession(chatID)
}

func playSong(chatID int64, message *telegram.NewMessage, song Song) error {
	return playSongOpt(chatID, message, song, shouldStayInCall(chatID))
}

func playSongOpt(chatID int64, message *telegram.NewMessage, song Song, stayInCall bool) error {
	holdSwitch(chatID)
	defer releaseSwitch(chatID)

	loading := wrapBQ("<b>" + smallcaps("loading") + "...</b>\n" + richEsc(shortTitle(song.Title, 40)))
	if message != nil {
		_ = editHTML(message, loading, nil)
	} else {
		message, _ = sendHTML(Bot, chatID, loading, nil)
	}

	if t := oembedTitle(extractVideoID(song.URL)); t != "" && (song.Title == "" || song.Title == "YouTube Video") {
		song.Title = t
	}
	setSeekState(chatID, 0)
	apMarkPlayed(chatID, extractVideoID(song.URL))

	src, err := downloadWithRetry(song.URL, song.Video)
	if err != nil {
		removeFromQueue(chatID, 0)
		_, _ = sendHTML(Bot, chatID, wrapBQ(smallcaps("download failed")+"\n<code>"+richEsc(err.Error())+"</code>"), nil)
		return err
	}
	if src.File != "" && !isHTTP(src.File) && song.Video && !fileHasVideo(src.File) {
		removeFromQueue(chatID, 0)
		_, _ = sendHTML(Bot, chatID, wrapBQ(smallcaps("vplay failed")+"\n"+smallcaps("api gave audio file, not video stream.")), nil)
		return fmt.Errorf("no video track in %s", src.File)
	}
	if src.File != "" && !isHTTP(src.File) && !song.Video {
		if p, e := maybeApplyEffects(chatID, src.File); e == nil {
			src.File, src.Audio = p, p
		}
		if secs := probeDuration(src.File); secs > 2 {
			song.DurationSeconds = int(secs)
			song.Duration = formatClock(secs)
		}
	}
	playPath := src.File
	if playPath == "" {
		playPath = src.Audio
	}
	setCurrentPath(chatID, playPath)

	media := buildMediaAV(src.Audio, src.Video, song.Video, 0)
	startErr := ntgPlay(chatID, media, song.Video)
	if startErr != nil {
		_, _ = sendHTML(Bot, chatID, wrapBQ(smallcaps("playback failed")+"\n<code>"+richEsc(startErr.Error())+"</code>"), nil)
		return startErr
	}

	markLiveSession(chatID)
	callIsVideo[chatID] = song.Video
	addServedChat(chatID)
	incrementPlayCount(chatID)
	total := float64(parseDur(song.Duration))
	if total <= 0 && song.DurationSeconds > 0 {
		total = float64(song.DurationSeconds)
	}
	if message != nil {
		_, _ = message.Delete()
	}
	msg := sendNowPlaying(chatID, song)
	go updateProgress(chatID, msg, time.Now(), total, song)
	return nil
}

func updateProgress(chatID int64, msg *telegram.NewMessage, start time.Time, total float64, song Song) {
	for {
		time.Sleep(15 * time.Second)
		if isSwitching(chatID) {
			continue
		}
		cur := peekCurrent(chatID)
		if cur == nil || cur.URL != song.URL {
			return
		}
		elapsed := time.Since(start).Seconds()
		if total > 0 && elapsed > total {
			elapsed = total
		}
		if msg != nil {
			_ = editHTML(msg, nowPlayingCaption(song), nowPlayingKB(chatID, elapsed, total))
		}
		if total > 0 && elapsed >= total {
			go handleStreamEnd(chatID)
			return
		}
	}
}
