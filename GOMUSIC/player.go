package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

func nowPlayingCaption(song Song) string {
	body := smallcaps("streaming in vc") + "\n\n" +
		smallcaps("title") + " : " + richEsc(smallcaps(shortTitle(song.Title, 42))) + "\n" +
		smallcaps("duration") + " : " + richEsc(smallcaps(song.Duration)) + "\n" +
		smallcaps("request by") + " : " + richEsc(song.Requester) + "\n\n" +
		smallcaps("powered by") + " : " + smallcaps("gomusic") + "\n" +
		smallcaps("yt music api powered by") + " : " + smallcaps("aruyt api")
	return wrapBQ(body)
}

func nowPlayingKB(elapsed, total float64) telegram.ReplyMarkup {
	return gogramMarkup(GetNowPlayingMarkup(progressBar(elapsed, total)))
}

func makePanelImage(cover, videoID string) string {
	if cover == "" {
		return ""
	}
	_ = os.MkdirAll(downloadDir, 0o755)
	name := "panel.jpg"
	if len(videoID) >= 6 {
		name = "panel_" + videoID + ".jpg"
	}
	out := filepath.Join(downloadDir, name)
	cmd := exec.Command("ffmpeg", "-y", "-i", cover,
		"-vf", "scale=1280:720:force_original_aspect_ratio=increase,crop=1280:720",
		"-frames:v", "1", "-q:v", "2", out)
	if err := cmd.Run(); err != nil {
		return cover
	}
	return out
}

func sendNowPlaying(chatID int64, song Song) *telegram.NewMessage {
	caption := nowPlayingCaption(song)
	kb := nowPlayingKB(0, float64(parseDur(song.Duration)))
	vid := extractVideoID(song.URL)
	cover := cacheThumb(thumbFor(song.URL, song.Thumbnail))
	thumb := makePanelImage(cover, vid)
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
	if hasLocalCall(chatID) || isLiveSession(chatID) {
		return true
	}
	_, ok := activeCalls[chatID]
	return ok
}

func playSong(chatID int64, message *telegram.NewMessage, song Song) error {
	return playSongOpt(chatID, message, song, shouldStayInCall(chatID))
}

func playSongOpt(chatID int64, message *telegram.NewMessage, song Song, stayInCall bool) error {
	holdSwitch(chatID)
	defer releaseSwitch(chatID)

	if shouldStayInCall(chatID) {
		stayInCall = true
	}

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

	if !stayInCall && !isLiveSession(chatID) {
		vcDone := make(chan struct{})
		go func() {
			_ = ensureVC(chatID)
			close(vcDone)
		}()
		select {
		case <-vcDone:
		case <-time.After(4 * time.Second):
		}
	}

	src, err := resolveDirectURL(song.URL, song.Video)
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
	var startErr error
	if stayInCall {
		startErr = ntgPlay(chatID, media, song.Video)
		if startErr != nil {
			log.Println("kanha play same-call:", startErr)
			startErr = ntgPlaySameCall(chatID, media)
			if startErr != nil {
				startErr = joinExistingCall(chatID, media, song.Video)
			}
		}
	} else {
		startErr = startNTGStreamWithMedia(chatID, media, song.Video)
		if startErr != nil && !isLiveSession(chatID) {
			_ = ensureVC(chatID)
			startErr = startNTGStreamWithMedia(chatID, media, song.Video)
		}
	}
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
			_ = editHTML(msg, nowPlayingCaption(song), nowPlayingKB(elapsed, total))
		}
		if total > 0 && elapsed >= total {
			go handleStreamEnd(chatID)
			return
		}
	}
}
