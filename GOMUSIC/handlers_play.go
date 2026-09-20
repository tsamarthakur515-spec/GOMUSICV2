package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

func handlePlay(m *telegram.NewMessage) error  { return processPlayCommand(m, false) }
func handleVPlay(m *telegram.NewMessage) error { return processPlayCommand(m, true) }

func processPlayCommand(m *telegram.NewMessage, video bool) error {
	if blocked(m) || m.IsPrivate() {
		return nil
	}
	chatID := m.ChatID()
	addServedChat(chatID)
	addServedUser(userIDOf(m))
	query := cmdArgs(m)
	_, _ = m.Delete()
	if last, ok := lastCmd[chatID]; ok && time.Since(last) < time.Duration(Cooldown)*time.Second {
		return nil
	}
	lastCmd[chatID] = time.Now()
	if query == "" {
		_, _ = sendHTML(Bot, chatID, richHeading("usage", 3)+richNote("<code>/play song name</code>"), nil)
		return nil
	}
	return processPlay(m, query, video)
}

func processPlay(m *telegram.NewMessage, query string, video bool) error {
	chatID := m.ChatID()
	pm, _ := sendHTML(Bot, chatID, richHeading("processing...", 3), nil)
	ok, banned := assistantIn(chatID)
	if banned {
		_ = editHTML(pm, richHeading("assistant banned", 3)+richNote("unban @"+assistantUsername), nil)
		return nil
	}
	if !ok {
		_ = editHTML(pm, richHeading("assistant is joining...", 3), nil)
		if !tryJoinAssistant(chatID, pm) {
			return nil
		}
	}
	_ = promoteAssistant(chatID)
	if strings.Contains(query, "youtu.be/") {
		if parts := strings.Split(query, "youtu.be/"); len(parts) > 1 {
			id := strings.Split(strings.Split(parts[1], "?")[0], "&")[0]
			query = "https://www.youtube.com/watch?v=" + id
		}
	}
	urlStr, title, durISO, thumb, playlist, err := searchSmart(query)
	if err != nil {
		_ = editHTML(pm, richHeading("search failed", 3)+richNote("<code>"+richEsc(err.Error())+"</code>"), nil)
		return nil
	}
	if urlStr == "" && len(playlist) == 0 {
		_ = editHTML(pm, richHeading("song not found", 3), nil)
		return nil
	}
	req := userNameOf(m)
	reqID := userIDOf(m)
	if len(playlist) > 0 {
		firstEmpty := queueSize(chatID) == 0
		for _, item := range playlist {
			addToQueue(chatID, Song{URL: item.Link, Title: item.Title, Duration: isoToHuman(item.Duration), DurationSeconds: isoToSec(item.Duration), Requester: req, RequesterID: reqID, Thumbnail: item.Thumbnail, Video: video})
		}
		if firstEmpty {
			if first := peekCurrent(chatID); first != nil {
				return playSong(chatID, pm, *first)
			}
		}
		return nil
	}
	song := Song{URL: urlStr, Title: title, Duration: isoToHuman(durISO), DurationSeconds: parseDur(durISO), Requester: req, RequesterID: reqID, Thumbnail: thumb, Video: video}
	pos := addToQueue(chatID, song)
	if pos == 1 {
		return playSong(chatID, pm, song)
	}
	_, _ = sendHTML(Bot, chatID, richHeading("added to queue", 3)+richNote(richEsc(title)), nil)
	if pm != nil {
		_, _ = pm.Delete()
	}
	return nil
}

func assistantIn(chatID int64) (present bool, banned bool) {
	me, err := Assistant.GetMe()
	if err != nil {
		return false, false
	}
	member, err := Bot.GetChatMember(chatID, me.ID)
	if err == nil && member != nil {
		st := strings.ToLower(fmt.Sprint(member.Status))
		if strings.Contains(st, "ban") || strings.Contains(st, "kicked") {
			return false, true
		}
		if strings.Contains(st, "left") {
			return false, false
		}
		return true, false
	}
	_, err = Assistant.GetChat(chatID)
	return err == nil, false
}

func assistantInput() (*telegram.InputUserObj, error) {
	me, err := Assistant.GetMe()
	if err != nil {
		return nil, err
	}
	return &telegram.InputUserObj{UserID: me.ID, AccessHash: me.AccessHash}, nil
}

func inviteHashFromLink(link string) string {
	link = strings.TrimSpace(link)
	for _, p := range []string{"https://t.me/+", "http://t.me/+", "https://telegram.me/+", "https://t.me/joinchat/", "http://t.me/joinchat/", "t.me/+", "t.me/joinchat/"} {
		if strings.HasPrefix(strings.ToLower(link), strings.ToLower(p)) {
			return strings.Trim(link[len(p):], "/")
		}
	}
	return link
}

func exportInviteLink(chatID int64) (string, error) {
	inv, err := Bot.ExportInvite(chatID)
	if err != nil {
		return "", err
	}
	switch v := inv.(type) {
	case *telegram.ChatInviteExported:
		return v.Link, nil
	case *telegram.ChatInvitePublicJoinRequests:
		return "", fmt.Errorf("group has join requests enabled; turn that off")
	default:
		return "", fmt.Errorf("unexpected invite type %T", inv)
	}
}

func promoteAssistant(chatID int64) error {
	user, err := assistantInput()
	if err != nil {
		return err
	}
	peer, err := Bot.ResolvePeer(chatID)
	if err != nil {
		return err
	}
	rights := telegram.ChatAdminRights{
		DeleteMessages: true,
		InviteUsers:    true,
		PinMessages:    true,
		ManageCall:     true,
		Other:          true,
	}
	switch p := peer.(type) {
	case *telegram.InputPeerChannel:
		_, err = Bot.ChannelsEditAdmin(&telegram.InputChannelObj{ChannelID: p.ChannelID, AccessHash: p.AccessHash}, user, &rights, "assistant")
		return err
	case *telegram.InputPeerChat:
		_, err = Bot.MessagesEditChatAdmin(p.ChatID, user, true)
		return err
	default:
		return nil
	}
}

func tryJoinAssistant(chatID int64, pm *telegram.NewMessage) bool {
	link, err := exportInviteLink(chatID)
	if err != nil {
		_ = editHTML(pm, richHeading("assistant join failed", 3)+
			richNote("Give the bot Invite Users permission, or add <b>@"+assistantUsername+"</b> manually.")+
			richNote("<code>"+richEsc(err.Error())+"</code>"), nil)
		return false
	}
	hash := inviteHashFromLink(link)
	var joinErr error
	if _, joinErr = Assistant.JoinChannel(link); joinErr != nil {
		low := strings.ToLower(joinErr.Error())
		if strings.Contains(low, "already") {
			_ = promoteAssistant(chatID)
			return true
		}
		if hash != "" {
			if _, err2 := Assistant.MessagesImportChatInvite(hash); err2 == nil || strings.Contains(strings.ToLower(err2.Error()), "already") {
				time.Sleep(time.Second)
				_ = promoteAssistant(chatID)
				return true
			} else {
				joinErr = err2
			}
		}
		_ = editHTML(pm, richHeading("assistant join failed", 3)+
			richNote("Telegram bots cannot add users with ChannelsInviteToChannel.")+richNote("Add <b>@"+assistantUsername+"</b> to the group once, give Manage Video Chats, then /play.")+
			richNote("<code>"+richEsc(joinErr.Error())+"</code>"), nil)
		return false
	}
	time.Sleep(time.Second)
	_ = promoteAssistant(chatID)
	return true
}

func handlePause(m *telegram.NewMessage) error {
	if blocked(m) || m.IsPrivate() || !isAuthorized(m) {
		return nil
	}
	_, _ = Calls.Pause(m.ChatID())
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("stream paused", 3), nil)
	return nil
}

func handleResume(m *telegram.NewMessage) error {
	if blocked(m) || m.IsPrivate() || !isAuthorized(m) {
		return nil
	}
	_, _ = Calls.Resume(m.ChatID())
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("stream resumed", 3), nil)
	return nil
}

func handleSkip(m *telegram.NewMessage) error {
	if blocked(m) || m.IsPrivate() || !isAuthorized(m) {
		return nil
	}
	chatID := m.ChatID()
	skipped := popCurrent(chatID)
	_ = Calls.Stop(chatID)
	time.Sleep(2 * time.Second)
	if skipped != nil {
		deleteFile(skipped.FilePath)
	}
	nxt := peekCurrent(chatID)
	if nxt != nil {
		dm, _ := sendHTML(Bot, chatID, richHeading("next track", 3), nil)
		return playSong(chatID, dm, *nxt)
	}
	_, _ = sendHTML(Bot, chatID, richHeading("queue empty", 3), nil)
	return nil
}

func handleStop(m *telegram.NewMessage) error {
	if blocked(m) || m.IsPrivate() || !isAuthorized(m) {
		return nil
	}
	leaveVC(m.ChatID())
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("playback stopped", 3), nil)
	return nil
}

func handleClear(m *telegram.NewMessage) error {
	if blocked(m) || m.IsPrivate() || !isAuthorized(m) {
		return nil
	}
	stopAutoplay(m.ChatID())
	clearQueue(m.ChatID())
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("queue cleared", 3), nil)
	return nil
}

func handleReboot(m *telegram.NewMessage) error {
	leaveVC(m.ChatID())
	_, _ = sendHTML(Bot, m.ChatID(), richHeading("chat rebooted", 3), nil)
	return nil
}
