package main

import (
	"fmt"
	"html"
	"strings"
)

const (
	btnAddMe     = " ᴇᴅᴅ мє тσ уσᴜр грσᴜп"
	btnHelpStart = " нєʟп & сσммᴇηдs"
	btnUpdates   = " ᴜпдᴇтєs"
	btnSupport   = "sᴜппσрт"
	btnSource    = " sσᴜрсє сσдє"
	btnCommands  = " сσммᴇηдs"
	btnClose     = "сʟσsє"
	btnBack      = " бᴇск"
	btnAdmin     = "ᴇдмɪη"
	btnAuth      = "ᴇᴜтн"
	btnBcast     = "б-сᴇsт"
	btnPlay      = "пʟᴇу"
	btnSudo      = "sᴜдσ"
	btnRestrict  = "рєsтрɪст"
	btnThumb     = "тнᴜмбηᴇɪʟ"
	btnStart     = "sтᴇрт"
	btnAutoplay  = "ᴇᴜтσпʟᴇу"
	btnPlaylist  = "пʟᴇуʟɪsт"
	btnVCLogs    = "вс-ʟσгs"
	btnInline    = "ɪηʟɪηє"
	btnPlayNow   = "пʟᴇу ησв"
	btnSkip      = "sкɪп"
	btnSeekBack  = "-15s"
	btnSeekFwd   = "+15s"
)

func startPrivateHTML(uid int64, name, bot string) string {
	user := fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, uid, richEsc(name))
	return "<blockquote><b>сᴇʟᴜтᴇтɪσηs</b> " + user + ",</blockquote>\n" +
		"<blockquote expandable><b>✧ ᴡᴇʟсσмє тσ " + richEsc(bot) + " — ᴇ пσᴡᴇрғᴜʟ ᴇηд нɪгн-спᴇᴇд тг мᴜсɪс бσт</b>\n" +
		"<b>✧ бᴜɪʟт ғσр смσσтн • стᴇбʟє • ʟᴇг-ғрᴇᴇ мᴜсɪс стрᴇᴇмɪηг</b>\n" +
		"<b>✧ пσᴡᴇрᴇд бу ᴇη σптɪмɪзᴇд уσᴜтᴜбᴇ ᴇпɪ ғσр ɪηsтᴇηт пʟᴇубᴇск</b>\n" +
		"<b>✧ ᴇηжσу нɪгн ǫᴜᴇʟɪту ᴇᴜдɪσ ᴡɪтн sᴇᴇмʟᴇss сσηтрσʟ</b>\n" +
		"<b>•──────────────•</b>\n" +
		"<b>✧ ᴜсᴇ нᴇʟп тσ вɪᴇᴡ ᴇʟʟ сσммᴇηдs ᴇηд ғᴇᴇтᴜрᴇs</b></blockquote>"
}

func startGroupHTML() string {
	return "<blockquote><b>💫 ɪ’м нєрє!</b> тру ᴇ сσммᴇηд…</blockquote>"
}

func helpMainHTML() string {
	return "<blockquote><b>🔮 єxпʟσрє тнє сσмпʟєтє сσммᴇηд ɪηдєx бєʟσᴡ</b></blockquote>\n\n" +
		"<blockquote><b>• ᴇссєss єxпєрт тєснɪсᴇʟ гᴜɪдᴇηсє &amp; рєᴇʟ-тɪмє sᴜппσрт</b></blockquote>\n" +
		"<blockquote><b>• єxєсᴜтє ᴇʟʟ сσммᴇηдs ᴜсɪηг sтᴇηдᴇрд прєғɪx ➜</b></blockquote>"
}

func helpPrivateOnlyHTML() string {
	return "<blockquote><b>нɪ! ғσр бσт нєʟп ᴇηд сσммᴇηдs, пʟєᴇsє дм мє дɪрєстʟу</b></blockquote>"
}

func streamNowPlayingHTML(song Song) string {
	link := song.URL
	if strings.TrimSpace(link) == "" {
		link = "#"
	}
	return "<blockquote><b>💮 пʟᴇубᴇск ᴇстɪвᴇтєд. | єηжσу тнє мᴜсɪс |</b></blockquote>\n" +
		"<blockquote expandable>▫ <b>мєʟσду :</b> <a href=\"" + html.EscapeString(link) + "\">" + richEsc(shortTitle(song.Title, 48)) + "</a>\n" +
		"▫ <b>ʟєηгтн :</b> " + richEsc(song.Duration) + "\n" +
		"▫ <b>рєǫᴜєsтєр :</b> " + richEsc(song.Requester) + "</blockquote>"
}

func incomingTrackHTML(index int, song Song) string {
	link := song.URL
	if strings.TrimSpace(link) == "" {
		link = "#"
	}
	return "<blockquote><b>💮 ɪηсσмɪηг трᴇск дєтєстєд : #" + fmt.Sprintf("%d", index) + "</b></blockquote>\n" +
		"<blockquote expandable><b>🎋 мєʟσду :</b> <a href=\"" + html.EscapeString(link) + "\">" + richEsc(shortTitle(song.Title, 35)) + "</a>\n" +
		"<b>✨ ʟєηгтн :</b> " + richEsc(song.Duration) + "\n" +
		"<b>🌿 рєǫᴜєsтєр :</b> " + richEsc(song.Requester) + "\n\n" +
		"💐 sтᴇηдбу, уσᴜр sєssɪη бєгɪηs sнσртʟу</blockquote>"
}

func autoplayBtnText(on bool) string {
	state := "дɪsᴇбʟєд"
	if on {
		state = "єηᴇбʟєд"
	}
	return "♫ ᴇᴜтσпʟᴇу: " + state
}

func queueListHTML(q []Song) string {
	if len(q) == 0 {
		return "<blockquote><b>ησ ᴇстɪвє пʟᴇубᴇск.</b>\n<i>ησтнɪηг ɪs ǫᴜєᴜєд рɪгнт ησᴡ.</i></blockquote>"
	}
	var b strings.Builder
	b.WriteString("<blockquote><b>🎶 сᴜррєηт ǫᴜєᴜє</b></blockquote>\n\n")
	b.WriteString("<b>▶️ ησᴡ пʟᴇуɪηг:</b>\n")
	cur := q[0]
	fmt.Fprintf(&b, "🎧 <a href=\"%s\">%s</a> — %s [%s]\n\n",
		html.EscapeString(cur.URL), richEsc(shortTitle(cur.Title, 35)), richEsc(cur.Requester), richEsc(cur.Duration))
	if len(q) == 1 {
		b.WriteString("<i>🗭 ησ мσрє sηгs ɪη ǫᴜєᴜє.</i>")
		return b.String()
	}
	rest := q[1:]
	useQuote := len(rest) >= 3
	b.WriteString("<b>⏭️ ᴜп ηєxт:</b>\n")
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
		fmt.Fprintf(&b, "\n… ᴇηд %d мσрє", extra)
	}
	return b.String()
}

var kanhaHelp = map[string]string{
	"admin":    "<blockquote><b>/pause /resume /skip /stop /queue /clear /seek /speed</b></blockquote>",
	"auth":     "<blockquote><b>ᴇᴜтн ᴜсєрs сᴇη мᴇηᴇгє стрєᴇмs.</b></blockquote>",
	"bcast":    "<blockquote><b>/broadcast</b> : sєηд тσ ᴇʟʟ sєрвєд снᴇтs.</blockquote>",
	"play":     "<blockquote><b>/play</b> ᴇᴜдɪσ • <b>/vplay</b> вɪдєσ • <b>/queue</b></blockquote>",
	"sudo":     "<blockquote><b>/stats /reboot /broadcast</b></blockquote>",
	"restrict": "<blockquote><b>/gblock /gunblock /ublock /uunblock /blocklist</b></blockquote>",
	"thumb":    "<blockquote><b>пʟᴇуєр пᴇηєʟ сσвєр ᴇрт ᴇᴜтσ снσᴡs.</b></blockquote>",
	"start":    "<blockquote><b>/start /help /ping /stats /id</b></blockquote>",
	"autoplay": "<blockquote><b>/autoplay</b> тσггʟє ᴇᴜтσпʟᴇу.</blockquote>",
	"playlist": "<blockquote><b>/play</b> σр <b>/vplay</b> пє уσᴜтᴜбє пʟᴇуʟɪsт ʟɪηк.</blockquote>",
	"vclogs":   "<blockquote><b>пʟᴇубᴇск стᴇтᴜс пʟᴇуєр пᴇηєʟ пє.</b></blockquote>",
	"inline":   "<blockquote><b>▷ II ⟳ ‣‣I ▢ -15s +15s</b></blockquote>",
}
