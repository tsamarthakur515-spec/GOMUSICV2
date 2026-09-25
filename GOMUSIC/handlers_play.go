package main

import (
	"fmt"
	"regexp"
	"strconv"
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
		var first Song
		for i, item := range playlist {
			s := Song{URL: item.Link, Title: item.Title, Duration: isoToHuman(item.Duration), DurationSeconds: isoToSec(item.Duration), Requester: req, RequesterID: reqID, Thumbnail: item.Thumbnail, Video: video}
			if i == 0 {
				first = s
			}
			addToQueue(chatID, s)
		}
		if firstEmpty {
			return playSongOpt(chatID, pm, first, true)
		}
		_ = editHTML(pm, wrapBQ(smallcaps("playlist queued")+"\n"+fmt.Sprintf("%d", len(playlist))+" "+smallcaps("tracks")), gogramMarkup(GetQueuedMarkup(chatID, 1)))
		return nil
	}
	song := Song{URL: urlStr, Title: title, Duration: isoToHuman(durISO), DurationSeconds: parseDur(durISO), Requester: req, RequesterID: reqID, Thumbnail: thumb, Video: video}
	return RoomPlay(chatID, song, false, pm)
}

func skipCurrent(chatID int64) error {
	return RoomChangeStream(chatID)
}

func chatIDForms(chatID int64) []int64 {
	ids := []int64{chatID}
	s := strconv.FormatInt(chatID, 10)
	if strings.HasPrefix(s, "-100") {
		if raw, err := strconv.ParseInt(s[4:], 10, 64); err == nil && raw != 0 {
			ids = append(ids, raw, -raw)
		}
	}
	return ids
}

func alreadyMemberErr(err error) bool {
	if err == nil {
		return false
	}
	low := strings.ToLower(err.Error())
	return strings.Contains(low, "already") ||
		strings.Contains(low, "user_already_participant") ||
		strings.Contains(low, "already a participant") ||
		strings.Contains(low, "already_participant")
}

func floodWait(err error) time.Duration {
	if err == nil {
		return 0
	}
	s := err.Error()
	re := regexp.MustCompile(`(?i)FLOOD_WAIT_(\d+)`)
	if m := re.FindStringSubmatch(s); len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		if n > 0 && n < 120 {
			return time.Duration(n+1) * time.Second
		}
	}
	re = regexp.MustCompile(`(?i)wait (\d+) seconds`)
	if m := re.FindStringSubmatch(s); len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		if n > 0 && n < 120 {
			return time.Duration(n+1) * time.Second
		}
	}
	return 0
}

func withFloodWait(fn func() error) error {
	var err error
	for i := 0; i < 4; i++ {
		err = fn()
		if err == nil || alreadyMemberErr(err) {
			return nil
		}
		if d := floodWait(err); d > 0 {
			time.Sleep(d)
			continue
		}
		return err
	}
	return err
}

func assistantIn(chatID int64) (present bool, banned bool) {
	if Assistant == nil || Bot == nil {
		return false, false
	}
	me, err := Assistant.GetMe()
	if err != nil || me == nil {
		return false, false
	}
	for _, id := range chatIDForms(chatID) {
		member, err := Bot.GetChatMember(id, me.ID)
		if err == nil && member != nil {
			st := strings.ToLower(fmt.Sprint(member.Status))
			if strings.Contains(st, "ban") || strings.Contains(st, "kick") {
				return false, true
			}
			if strings.Contains(st, "left") {
				continue
			}
			return true, false
		}
		if _, err := Assistant.GetChat(id); err == nil {
			return true, false
		}
	}
	return false, false
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
	var last error
	for _, id := range chatIDForms(chatID) {
		inv, err := Bot.ExportInvite(id)
		if err != nil {
			last = err
			continue
		}
		switch v := inv.(type) {
		case *telegram.ChatInviteExported:
			return v.Link, nil
		case *telegram.ChatInvitePublicJoinRequests:
			return "", fmt.Errorf("group has join requests enabled; turn that off")
		default:
			last = fmt.Errorf("unexpected invite type %T", inv)
		}
	}
	if last == nil {
		last = fmt.Errorf("invite export failed")
	}
	return "", last
}

func promoteAssistant(chatID int64) error {
	user, err := assistantInput()
	if err != nil {
		return err
	}
	for _, id := range chatIDForms(chatID) {
		peer, err := Bot.ResolvePeer(id)
		if err != nil {
			continue
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
			if err == nil {
				return nil
			}
		case *telegram.InputPeerChat:
			_, err = Bot.MessagesEditChatAdmin(p.ChatID, user, true)
			if err == nil {
				return nil
			}
		}
	}
	return nil
}

func inviteAssistantViaBot(chatID int64) error {
	user, err := assistantInput()
	if err != nil {
		return err
	}
	var last error
	for _, id := range chatIDForms(chatID) {
		peer, err := Bot.ResolvePeer(id)
		if err != nil {
			last = err
			continue
		}
		switch p := peer.(type) {
		case *telegram.InputPeerChannel:
			_, err = Bot.ChannelsInviteToChannel(&telegram.InputChannelObj{ChannelID: p.ChannelID, AccessHash: p.AccessHash}, []telegram.InputUser{user})
		case *telegram.InputPeerChat:
			_, err = Bot.MessagesAddChatUser(p.ChatID, user, 0)
		default:
			last = fmt.Errorf("unsupported peer %T", peer)
			continue
		}
		if err == nil || alreadyMemberErr(err) {
			return nil
		}
		last = err
	}
	return last
}

func tryJoinAssistant(chatID int64, pm *telegram.NewMessage) bool {
	if ok, banned := assistantIn(chatID); banned {
		_ = editHTML(pm, wrapBQ(smallcaps("assistant banned")+"\n"+smallcaps("unban")+" @"+assistantUsername), nil)
		return false
	} else if ok {
		_ = promoteAssistant(chatID)
		return true
	}

	inviteErr := withFloodWait(func() error { return inviteAssistantViaBot(chatID) })
	if inviteErr == nil {
		time.Sleep(800 * time.Millisecond)
		_ = promoteAssistant(chatID)
		return true
	}

	link, err := exportInviteLink(chatID)
	if err == nil && link != "" {
		hash := inviteHashFromLink(link)
		joinErr := withFloodWait(func() error {
			_, e := Assistant.JoinChannel(link)
			return e
		})
		if joinErr != nil && hash != "" {
			joinErr = withFloodWait(func() error {
				_, e := Assistant.MessagesImportChatInvite(hash)
				return e
			})
		}
		if joinErr == nil {
			time.Sleep(800 * time.Millisecond)
			_ = promoteAssistant(chatID)
			return true
		}
		inviteErr = joinErr
	}

	time.Sleep(1200 * time.Millisecond)
	if ok, _ := assistantIn(chatID); ok {
		_ = promoteAssistant(chatID)
		return true
	}

	msg := smallcaps("assistant join failed") + "\n" +
		smallcaps("add") + " @" + assistantUsername + " " + smallcaps("once then play")
	if inviteErr != nil && floodWait(inviteErr) == 0 {
		msg += "\n<code>" + richEsc(inviteErr.Error()) + "</code>"
	}
	_ = editHTML(pm, wrapBQ(msg), nil)
	return false
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
	leaveVCNow(m.ChatID())
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
	_, _ = sendHTML(Bot, m.ChatID(), wrapBQ(b.String()), gogramMarkup(GetQueuedMarkup(m.ChatID(), 1)))
	return nil
}

func handleReboot(m *telegram.NewMessage) error {
	leaveVCNow(m.ChatID())
	_, _ = sendHTML(Bot, m.ChatID(), wrapBQ(smallcaps("chat rebooted")), nil)
	return nil
}
