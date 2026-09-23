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
	already := isBusy(chatID)
	var pm *telegram.NewMessage
	if already {
		pm, _ = sendHTML(Bot, chatID, wrapBQ(smallcaps("adding to queue...")), nil)
	} else {
		pm, _ = sendHTML(Bot, chatID, wrapBQ(smallcaps("processing...")), nil)
		ok, banned := assistantIn(chatID)
		if banned {
			_ = editHTML(pm, wrapBQ(smallcaps("assistant banned")+"\n"+smallcaps("unban")+" @"+assistantUsername), nil)
			return nil
		}
		if !ok {
			_ = editHTML(pm, wrapBQ(smallcaps("assistant is joining...")), nil)
			if !tryJoinAssistant(chatID, pm) {
				return nil
			}
		}
		_ = promoteAssistant(chatID)
		go func() { _ = ensureVC(chatID) }()
	}
	if strings.Contains(query, "youtu.be/") {
		if parts := strings.Split(query, "youtu.be/"); len(parts) > 1 {
			id := strings.Split(strings.Split(parts[1], "?")[0], "&")[0]
			query = "https://www.youtube.com/watch?v=" + id
		}
	}
	urlStr, title, durISO, thumb, playlist, err := searchSmart(query)
	if err != nil {
		_ = editHTML(pm, wrapBQ(smallcaps("search failed")+"\n<code>"+richEsc(err.Error())+"</code>"), nil)
		return nil
	}
	if urlStr == "" && len(playlist) == 0 {
		_ = editHTML(pm, wrapBQ(smallcaps("song not found")), nil)
		return nil
	}
	req := userNameOf(m)
	reqID := userIDOf(m)
	if len(playlist) > 0 {
		firstEmpty := !already && queueSize(chatID) == 0
		for _, item := range playlist {
			addToQueue(chatID, Song{URL: item.Link, Title: item.Title, Duration: isoToHuman(item.Duration), DurationSeconds: isoToSec(item.Duration), Requester: req, RequesterID: reqID, Thumbnail: item.Thumbnail, Video: video})
		}
		if firstEmpty {
			if first := peekCurrent(chatID); first != nil {
				return playSong(chatID, pm, *first)
			}
		}
		_ = editHTML(pm, wrapBQ(smallcaps("playlist queued")+"\n"+fmt.Sprintf("%d", len(playlist))+" "+smallcaps("tracks")), gogramMarkup(GetQueuedMarkup(1)))
		return nil
	}
	song := Song{URL: urlStr, Title: title, Duration: isoToHuman(durISO), DurationSeconds: parseDur(durISO), Requester: req, RequesterID: reqID, Thumbnail: thumb, Video: video}
	pos := addToQueue(chatID, song)
	if !already && pos == 1 {
		return playSong(chatID, pm, song)
	}
	body := smallcaps("added to queue") + "\n\n" +
		smallcaps("title") + " : " + richEsc(shortTitle(title, 42)) + "\n" +
		smallcaps("duration") + " : " + richEsc(isoToHuman(durISO)) + "\n" +
		smallcaps("position") + " : " + fmt.Sprintf("%d", pos)
	_ = editHTML(pm, wrapBQ(body), gogramMarkup(GetQueuedMarkup(pos-1)))
	return nil
}

func skipCurrent(chatID int64) error {
	beginSwitch(chatID)
	stay := shouldStayInCall(chatID)
	skipped := popCurrent(chatID)
	if skipped != nil {
		deleteFile(skipped.FilePath)
	}
	nxt := peekCurrent(chatID)
	if nxt == nil {
		return fmt.Errorf("queue empty")
	}
	return playSongOpt(chatID, nil, *nxt, stay)
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
		_ = editHTML(pm, wrapBQ(smallcaps("assistant join failed")+"\n"+
			smallcaps("give invite users permission or add")+" @"+assistantUsername),
			nil)
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
				_ = promoteAssistant(chatID)
				return true
			} else {
				joinErr = err2
			}
		}
		_ = editHTML(pm, wrapBQ(smallcaps("assistant join failed")+"\n"+
			smallcaps("add")+" @"+assistantUsername+" "+smallcaps("once then play")+"\n<code>"+richEsc(joinErr.Error())+"</code>"), nil)
		return false
	}
	_ = promoteAssistant(chatID)
	return true
}

func handlePause(m *telegram.NewMessage) error {
	if blocked(m) || m.IsPrivate() || !isAuthorized(m) {
		return nil
	}
	_, _ = Calls.Pause(m.ChatID())
	_, _ = sendHTML(Bot, m.ChatID(), wrapBQ(smallcaps("stream paused")), nil)
	return nil
}

func handleResume(m *telegram.NewMessage) error {
	if blocked(m) || m.IsPrivate() || !isAuthorized(m) {
		return nil
	}
	_, _ = Calls.Resume(m.ChatID())
	_, _ = sendHTML(Bot, m.ChatID(), wrapBQ(smallcaps("stream resumed")), nil)
	return nil
}

func handleSkip(m *telegram.NewMessage) error {
	if blocked(m) || m.IsPrivate() || !isAuthorized(m) {
		return nil
	}
	if err := skipCurrent(m.ChatID()); err != nil {
		_, _ = sendHTML(Bot, m.ChatID(), wrapBQ(smallcaps("queue empty")), nil)
		return nil
	}
	return nil
}

func handleStop(m *telegram.NewMessage) error {
	if blocked(m) || m.IsPrivate() || !isAuthorized(m) {
		return nil
	}
	leaveVC(m.ChatID())
	_, _ = sendHTML(Bot, m.ChatID(), wrapBQ(smallcaps("playback stopped")), nil)
	return nil
}

func handleClear(m *telegram.NewMessage) error {
	if blocked(m) || m.IsPrivate() || !isAuthorized(m) {
		return nil
	}
	stopAutoplay(m.ChatID())
	clearQueue(m.ChatID())
	_, _ = sendHTML(Bot, m.ChatID(), wrapBQ(smallcaps("queue cleared")), nil)
	return nil
}

func handleQueue(m *telegram.NewMessage) error {
	if blocked(m) || m.IsPrivate() {
		return nil
	}
	q := getQueue(m.ChatID())
	if len(q) == 0 {
		_, _ = sendHTML(Bot, m.ChatID(), wrapBQ(smallcaps("queue is empty")), nil)
		return nil
	}
	var b strings.Builder
	b.WriteString(smallcaps("queue") + "\n\n")
	for i, s := range q {
		mark := smallcaps("next")
		if i == 0 {
			mark = smallcaps("now")
		}
		b.WriteString(fmt.Sprintf("%d. %s\n%s : %s\n", i+1, richEsc(shortTitle(s.Title, 36)), mark, richEsc(s.Duration)))
	}
	_, _ = sendHTML(Bot, m.ChatID(), wrapBQ(b.String()), gogramMarkup(GetQueuedMarkup(1)))
	return nil
}

func handleReboot(m *telegram.NewMessage) error {
	leaveVC(m.ChatID())
	_, _ = sendHTML(Bot, m.ChatID(), wrapBQ(smallcaps("chat rebooted")), nil)
	return nil
}
