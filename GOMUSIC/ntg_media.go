package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/nikhil390u8o/GOMUSICV2/ntgcalls"
)

func videoSize(path string) (int, int) {
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", path).Output()
	if err != nil {
		return 1280, 720
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(parts) != 2 {
		return 1280, 720
	}
	w, _ := strconv.Atoi(parts[0])
	h, _ := strconv.Atoi(parts[1])
	if w <= 0 || h <= 0 {
		return 1280, 720
	}
	maxW, maxH := 1280, 720
	ratio := float64(w) / float64(h)
	nw := w
	if nw > maxW {
		nw = maxW
	}
	nh := int(float64(nw) / ratio)
	if nh > maxH {
		nh = maxH
		nw = int(float64(nh) * ratio)
	}
	if nw%2 != 0 {
		nw--
	}
	if nh%2 != 0 {
		nh--
	}
	if nw < 2 {
		nw = 2
	}
	if nh < 2 {
		nh = 2
	}
	return nw, nh
}

func buildMedia(path string, video bool, seekSec int) ntgcalls.MediaDescription {
	q := fmt.Sprintf("%q", path)
	seek := ""
	if seekSec > 0 {
		seek = fmt.Sprintf("-ss %d ", seekSec)
	}
	audio := &ntgcalls.AudioDescription{
		MediaSource:  ntgcalls.MediaSourceShell,
		SampleRate:   48000,
		ChannelCount: 2,
		Input:        fmt.Sprintf("ffmpeg %s-i %s -f s16le -ac 2 -ar 48000 -v quiet pipe:1", seek, q),
	}
	if !video {
		return ntgcalls.MediaDescription{Microphone: audio}
	}
	w, h := videoSize(path)
	cam := &ntgcalls.VideoDescription{
		MediaSource: ntgcalls.MediaSourceShell,
		Width:       int16(w),
		Height:      int16(h),
		Fps:         30,
		Input:       fmt.Sprintf("ffmpeg %s-i %s -f rawvideo -r 30 -pix_fmt yuv420p -vf scale=%d:%d -v quiet pipe:1", seek, q, w, h),
	}
	return ntgcalls.MediaDescription{Microphone: audio, Camera: cam}
}

func buildMediaFromURL(youtubeURL string) ntgcalls.MediaDescription {
	q := fmt.Sprintf("%q", youtubeURL)
	ytdlpCmd := fmt.Sprintf(
		"yt-dlp --no-warnings --no-check-certificates --extractor-args youtube:player_client=mweb,android_music -f bestaudio/best -o - %s | ffmpeg -i pipe:0 -f s16le -ac 2 -ar 48000 -v quiet pipe:1",
		q,
	)
	audio := &ntgcalls.AudioDescription{
		MediaSource:  ntgcalls.MediaSourceShell,
		SampleRate:   48000,
		ChannelCount: 2,
		Input:        ytdlpCmd,
	}
	return ntgcalls.MediaDescription{Microphone: audio}
}

func startNTGStreamWithMedia(chatID int64, media ntgcalls.MediaDescription, video bool) error {
	if Calls == nil {
		return fmt.Errorf("ntgcalls not ready")
	}
	if Calls.Calls()[chatID] != nil {
		return Calls.SetStreamSources(chatID, ntgcalls.CaptureStream, media)
	}
	return startNTGStreamPrebuilt(chatID, media, video)
}

func startNTGStreamPrebuilt(chatID int64, media ntgcalls.MediaDescription, video bool) error {
	if Calls == nil {
		return fmt.Errorf("ntgcalls not ready")
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
		activeCalls[chatID] = inputCall
		callIsVideo[chatID] = video
		return nil
	}
	if last == nil {
		last = fmt.Errorf("join call failed")
	}
	return last
}

func ytDuration(videoID string) float64 {
	out, err := exec.Command(
		"yt-dlp", "--no-warnings", "--print", "duration",
		"https://www.youtube.com/watch?v="+videoID,
	).Output()
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(out))
	var secs float64
	fmt.Sscanf(s, "%f", &secs)
	return secs
}
