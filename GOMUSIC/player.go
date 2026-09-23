package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/nikhil390u8o/GOMUSICV2/ntgcalls"
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
	if hasLocalCall(chatID) {
		return true
	}
	_, ok := activeCalls[chatID]
	return ok
}

func playSong(chatID int64, message *telegram.NewMessage, song Song) error {
	return playSongOpt(chatID, message, song, shouldStayInCall(chatID))
}

func playSongOpt(chatID int64, message *telegram.NewMessage, song Song, stayInCall bool) error {
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

	if !stayInCall {
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
	if stayInCall && Calls != nil {
		bumpStream(chatID)
		startErr = Calls.SetStreamSources(chatID, ntgcalls.CaptureStream, media)
		if startErr != nil && hasLocalCall(chatID) {
			log.Println("set stream sources:", startErr)
		}
	} else {
		startErr = startNTGStreamWithMedia(chatID, media, song.Video)
	}
	if startErr != nil && !stayInCall {
		low := strings.ToLower(startErr.Error())
		if strings.Contains(low, "no active") || strings.Contains(low, "groupcall") {
			_ = ensureVC(chatID)
			startErr = startNTGStreamWithMedia(chatID, media, song.Video)
		}
	}
	if startErr != nil && !stayInCall {
		removeFromQueue(chatID, 0)
		_, _ = sendHTML(Bot, chatID, wrapBQ(smallcaps("playback failed")+"\n<code>"+richEsc(startErr.Error())+"</code>"), nil)
		return startErr
	}
	if startErr != nil && stayInCall {
		_, _ = sendHTML(Bot, chatID, wrapBQ(smallcaps("track switch failed")+"\n<code>"+richEsc(startErr.Error())+"</code>"), nil)
		return startErr
	}

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
		time.Sleep(30 * time.Second)
		elapsed := time.Since(start).Seconds()
		if total > 0 && elapsed > total {
			elapsed = total
		}
		if msg != nil {
			_ = editHTML(msg, nowPlayingCaption(song), nowPlayingKB(elapsed, total))
		}
		if total > 0 && elapsed >= total {
			return
		}
	}
}
