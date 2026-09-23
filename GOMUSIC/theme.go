package main

import (
	"fmt"
	"html"
	"strings"
)

var (
	btnAddMe     = " " + smallcaps("add me to your group")
	btnHelpStart = " " + smallcaps("help & commands")
	btnUpdates   = " " + smallcaps("updates")
	btnSupport   = smallcaps("support")
	btnSource    = " " + smallcaps("source code")
	btnCommands  = " " + smallcaps("commands")
	btnClose     = smallcaps("close")
	btnBack      = " " + smallcaps("back")
	btnAdmin     = smallcaps("admin")
	btnAuth      = smallcaps("auth")
	btnBcast     = smallcaps("b-cast")
	btnPlay      = smallcaps("play")
	btnSudo      = smallcaps("sudo")
	btnRestrict  = smallcaps("restrict")
	btnThumb     = smallcaps("thumbnail")
	btnStart     = smallcaps("start")
	btnAutoplay  = smallcaps("autoplay")
	btnPlaylist  = smallcaps("playlist")
	btnVCLogs    = smallcaps("vc-logs")
	btnInline    = smallcaps("inline")
	btnPlayNow   = smallcaps("play now")
	btnSkip      = smallcaps("skip")
	btnSeekBack  = "-15s"
	btnSeekFwd   = "+15s"
)

func startPrivateHTML(uid int64, name, bot string) string {
	user := fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, uid, richEsc(name))
	return "<blockquote><b>" + smallcaps("salutations") + "</b> " + user + ",</blockquote>\n" +
		"<blockquote expandable><b>✧ " + smallcaps("welcome to") + " " + richEsc(bot) + " — " + smallcaps("a powerful and high-speed tg music bot") + "</b>\n" +
		"<b>✧ " + smallcaps("built for smooth stable lag-free music streaming") + "</b>\n" +
		"<b>✧ " + smallcaps("powered by an optimized youtube api for instant playback") + "</b>\n" +
		"<b>✧ " + smallcaps("enjoy high quality audio with seamless control") + "</b>\n" +
		"<b>•──────────────•</b>\n" +
		"<b>✧ " + smallcaps("use help to view all commands and features") + "</b></blockquote>"
}

func startGroupHTML() string {
	return "<blockquote><b>💫 " + smallcaps("i am here") + "!</b> " + smallcaps("try a command") + "</blockquote>"
}

func helpMainHTML() string {
	return "<blockquote><b>🔮 " + smallcaps("explore the complete command index below") + "</b></blockquote>\n\n" +
		"<blockquote><b>• " + smallcaps("access expert technical guidance and realtime support") + "</b></blockquote>\n" +
		"<blockquote><b>• " + smallcaps("execute all commands using standard prefix") + " ➜</b></blockquote>"
}

func helpPrivateOnlyHTML() string {
	return "<blockquote><b>" + smallcaps("for bot help and commands, please dm me directly") + "</b></blockquote>"
}

func streamNowPlayingHTML(song Song) string {
	link := song.URL
	if strings.TrimSpace(link) == "" {
		link = "#"
	}
	return "<blockquote><b>💮 " + smallcaps("playback activated") + " | " + smallcaps("enjoy the music") + " |</b></blockquote>\n" +
		"<blockquote expandable>▫ <b>" + smallcaps("melody") + " :</b> <a href=\"" + html.EscapeString(link) + "\">" + richEsc(shortTitle(song.Title, 48)) + "</a>\n" +
		"▫ <b>" + smallcaps("length") + " :</b> " + richEsc(song.Duration) + "\n" +
		"▫ <b>" + smallcaps("requester") + " :</b> " + richEsc(song.Requester) + "</blockquote>"
}

func incomingTrackHTML(index int, song Song) string {
	link := song.URL
	if strings.TrimSpace(link) == "" {
		link = "#"
	}
	return "<blockquote><b>💮 " + smallcaps("incoming track detected") + " : #" + fmt.Sprintf("%d", index) + "</b></blockquote>\n" +
		"<blockquote expandable><b>🎋 " + smallcaps("melody") + " :</b> <a href=\"" + html.EscapeString(link) + "\">" + richEsc(shortTitle(song.Title, 35)) + "</a>\n" +
		"<b>✨ " + smallcaps("length") + " :</b> " + richEsc(song.Duration) + "\n" +
		"<b>🥀 " + smallcaps("requester") + " :</b> " + richEsc(song.Requester) + "\n\n" +
		"💐 " + smallcaps("standby, your session begins shortly") + "</blockquote>"
}

func autoplayBtnText(on bool) string {
	state := smallcaps("disabled")
	if on {
		state = smallcaps("enabled")
	}
	return "♫ " + smallcaps("autoplay") + ": " + state
}

func queueListHTML(q []Song) string {
	if len(q) == 0 {
		return "<blockquote><b>" + smallcaps("no active playback") + ".</b>\n<i>" + smallcaps("nothing is queued right now") + ".</i></blockquote>"
	}
	var b strings.Builder
	b.WriteString("<blockquote><b>🎶 " + smallcaps("current queue") + "</b></blockquote>\n\n")
	b.WriteString("<b>▶️ " + smallcaps("now playing") + ":</b>\n")
	cur := q[0]
	fmt.Fprintf(&b, "🎧 <a href=\"%s\">%s</a> — %s [%s]\n\n",
		html.EscapeString(cur.URL), richEsc(shortTitle(cur.Title, 35)), richEsc(cur.Requester), richEsc(cur.Duration))
	if len(q) == 1 {
		b.WriteString("<i>📭 " + smallcaps("no more songs in queue") + ".</i>")
		return b.String()
	}
	rest := q[1:]
	useQuote := len(rest) >= 3
	b.WriteString("<b>⏭️ " + smallcaps("up next") + ":</b>\n")
	if useQuote {
		b.WriteString("<blockquote>")
	} else {
		b.WriteString("\n")
	}
	for i, track := range rest {
		if i >= 10 {
			break
		}
		fmt.Fprintf(&b, "%d. 🎵 <a href=\"%s\">%s</a> — %s [%s]\n",
			i+1, html.EscapeString(track.URL), richEsc(shortTitle(track.Title, 35)), richEsc(track.Requester), richEsc(track.Duration))
	}
	if useQuote {
		b.WriteString("</blockquote>")
	}
	if extra := len(rest) - 10; extra > 0 {
		fmt.Fprintf(&b, "\n… "+smallcaps("and")+" %d "+smallcaps("more"), extra)
	}
	return b.String()
}

var kanhaHelp = map[string]string{}

func init() {
	kanhaHelp = map[string]string{
		"admin":    "<blockquote><b>/pause /resume /skip /stop /queue /clear /seek /speed</b></blockquote>",
		"auth":     "<blockquote><b>" + smallcaps("auth users can manage streams") + "</b></blockquote>",
		"bcast":    "<blockquote><b>/broadcast</b> : " + smallcaps("send to all served chats") + "</blockquote>",
		"play":     "<blockquote><b>/play</b> " + smallcaps("audio") + " • <b>/vplay</b> " + smallcaps("video") + " • <b>/queue</b></blockquote>",
		"sudo":     "<blockquote><b>/stats /reboot /broadcast</b></blockquote>",
		"restrict": "<blockquote><b>/gblock /gunblock /ublock /uunblock /blocklist</b></blockquote>",
		"thumb":    "<blockquote><b>" + smallcaps("player panel shows track cover art") + "</b></blockquote>",
		"start":    "<blockquote><b>/start /help /ping /stats /id</b></blockquote>",
		"autoplay": "<blockquote><b>/autoplay</b> " + smallcaps("toggle autoplay") + "</blockquote>",
		"playlist": "<blockquote><b>/play</b> " + smallcaps("or") + " <b>/vplay</b> " + smallcaps("with a youtube playlist link") + "</blockquote>",
		"vclogs":   "<blockquote><b>" + smallcaps("playback status is shown on the player panel") + "</b></blockquote>",
		"inline":   "<blockquote><b>▷ II ⟳ ‣‣I ▢ -15s +15s</b></blockquote>",
	}
}
