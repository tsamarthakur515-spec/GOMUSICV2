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

func isHTTP(src string) bool {
	s := strings.ToLower(strings.TrimSpace(src))
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func ffmpegPreFlags(src string) string {
	if !isHTTP(src) {
		return "-nostdin -hide_banner -fflags +genpts+discardcorrupt -err_detect ignore_err "
	}
	return "-nostdin -hide_banner -reconnect 1 -reconnect_streamed 1 -reconnect_on_network_error 1 -reconnect_delay_max 8 -fflags +genpts+discardcorrupt -err_detect ignore_err "
}

func videoSize(path string) (int, int) {
	if isHTTP(path) {
		return 854, 480
	}
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", path).Output()
	if err != nil {
		return 854, 480
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(parts) != 2 {
		return 854, 480
	}
	w, _ := strconv.Atoi(parts[0])
	h, _ := strconv.Atoi(parts[1])
	if w <= 0 || h <= 0 {
		return 854, 480
	}
	maxW, maxH := 854, 480
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
	return buildMediaAV(path, path, video, seekSec)
}

func buildMediaAV(audioSrc, videoSrc string, video bool, seekSec int) ntgcalls.MediaDescription {
	if audioSrc == "" {
		audioSrc = videoSrc
	}
	if videoSrc == "" {
		videoSrc = audioSrc
	}
	aq := fmt.Sprintf("%q", audioSrc)
	vq := fmt.Sprintf("%q", videoSrc)
	seek := ""
	if seekSec > 0 {
		seek = fmt.Sprintf("-ss %d ", seekSec)
	}
	audioCmd := fmt.Sprintf(
		"ffmpeg %s%s-i %s -vn -af apad=pad_dur=1.2 -f s16le -ac 2 -ar 48000 -v quiet pipe:1",
		ffmpegPreFlags(audioSrc), seek, aq,
	)
	audio := &ntgcalls.AudioDescription{
		MediaSource:  ntgcalls.MediaSourceShell,
		SampleRate:   48000,
		ChannelCount: 2,
		Input:        audioCmd,
	}
	if !video {
		return ntgcalls.MediaDescription{Microphone: audio}
	}
	w, h := videoSize(videoSrc)
	cam := &ntgcalls.VideoDescription{
		MediaSource: ntgcalls.MediaSourceShell,
		Width:       int16(w),
		Height:      int16(h),
		Fps:         24,
		Input: fmt.Sprintf(
			"ffmpeg %s%s-i %s -an -f rawvideo -r 24 -pix_fmt yuv420p -vf scale=%d:%d -v quiet pipe:1",
			ffmpegPreFlags(videoSrc), seek, vq, w, h,
		),
	}
	return ntgcalls.MediaDescription{Microphone: audio, Camera: cam}
}

func startNTGStreamWithMedia(chatID int64, media ntgcalls.MediaDescription, video bool) error {
	if Calls == nil {
		return fmt.Errorf("ntgcalls not ready")
	}
	if Calls.Calls()[chatID] != nil {
		bumpStream(chatID)
		return Calls.SetStreamSources(chatID, ntgcalls.CaptureStream, media)
	}
	return startNTGStreamPrebuilt(chatID, media, video)
}

func startNTGStreamPrebuilt(chatID int64, media ntgcalls.MediaDescription, video bool) error {
	if Calls == nil {
		return fmt.Errorf("ntgcalls not ready")
	}
	var last error
	for attempt := 1; attempt <= 3; attempt++ {
		inputCall, err := resolveActiveCall(chatID)
		if err != nil {
			if e := ensureVC(chatID); e == nil {
				inputCall, err = resolveActiveCall(chatID)
			}
			if err != nil {
				last = err
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
		}
		if Calls.Calls()[chatID] != nil {
			_ = Calls.Stop(chatID)
			time.Sleep(300 * time.Millisecond)
		}
		local, err := Calls.CreateCall(chatID)
		if err != nil {
			_ = Calls.Stop(chatID)
			last = err
			time.Sleep(800 * time.Millisecond)
			continue
		}
		if err := Calls.SetStreamSources(chatID, ntgcalls.CaptureStream, media); err != nil {
			_ = Calls.Stop(chatID)
			last = err
			time.Sleep(800 * time.Millisecond)
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
			if shouldRetryJoin(err) && attempt < 3 {
				time.Sleep(time.Duration(attempt+1) * time.Second)
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
			time.Sleep(800 * time.Millisecond)
			continue
		}
		activeCalls[chatID] = inputCall
		callIsVideo[chatID] = video
		bumpStream(chatID)
		return nil
	}
	if last == nil {
		last = fmt.Errorf("join call failed")
	}
	return last
}
