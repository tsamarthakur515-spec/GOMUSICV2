package main

func helpPage(key string) string {
	pages := map[string]string{
		"admin": "<blockquote><b>⊚ " + smallcaps("admin commands") + "</b></blockquote>\n" +
			"<blockquote expandable>➭ <b>/pause</b> — " + smallcaps("halt the active stream") + "\n" +
			"➭ <b>/resume</b> — " + smallcaps("restart paused playback") + "\n" +
			"➭ <b>/skip</b> — " + smallcaps("play the next queued track") + "\n" +
			"➭ <b>/stop</b> — " + smallcaps("end stream and empty queue") + "\n" +
			"➭ <b>/queue</b> — " + smallcaps("show upcoming tracks") + "\n" +
			"➭ <b>/clear</b> — " + smallcaps("clear queued songs") + "\n" +
			"➭ <b>/seek</b> / <b>/seekback</b> — " + smallcaps("jump in the track") + "\n" +
			"➭ <b>/speed</b> — " + smallcaps("change playback speed") + "</blockquote>",
		"auth": "<blockquote><b>⊚ " + smallcaps("auth users") + "</b></blockquote>\n" +
			"<blockquote expandable>Ὂe " + smallcaps("group admins can control playback.") + "\n" +
			"➭ " + smallcaps("owner can use every command") + "\n" +
			"➭ " + smallcaps("admins can pause resume skip stop") + "</blockquote>",
		"bcast": "<blockquote><b>⊚ " + smallcaps("broadcast") + "</b></blockquote>\n" +
			"<blockquote expandable>➭ <b>/broadcast</b> " + smallcaps("or") + " <b>/gcast</b>\n" +
			"Ὂe " + smallcaps("owner only. sends a message to all served chats.") + "</blockquote>",
		"play": "<blockquote><b>⊚ " + smallcaps("play commands") + "</b></blockquote>\n" +
			"<blockquote expandable>➭ <b>/play</b> — " + smallcaps("audio stream in vc") + "\n" +
			"➭ <b>/vplay</b> — " + smallcaps("video stream in vc") + "\n" +
			"➭ <b>/queue</b> — " + smallcaps("view queued tracks") + "\n" +
			"Ὂe " + smallcaps("if a song is already playing, the next request is added to queue.") + "</blockquote>",
		"sudo": "<blockquote><b>⊚ " + smallcaps("sudo / owner") + "</b></blockquote>\n" +
			"<blockquote expandable>➭ <b>/stats</b> — " + smallcaps("bot metrics") + "\n" +
			"➭ <b>/reboot</b> — " + smallcaps("reset this chat player") + "\n" +
			"➭ <b>/broadcast</b> — " + smallcaps("message all chats") + "\n" +
			"➭ <b>/gblock</b> / <b>/ublock</b> — " + smallcaps("restrict chats or users") + "</blockquote>",
		"restrict": "<blockquote><b>⊚ " + smallcaps("blacklist") + "</b></blockquote>\n" +
			"<blockquote expandable>➭ <b>/gblock</b> — " + smallcaps("block a group") + "\n" +
			"➭ <b>/gunblock</b> — " + smallcaps("unblock a group") + "\n" +
			"➭ <b>/ublock</b> — " + smallcaps("block a user") + "\n" +
			"➭ <b>/uunblock</b> — " + smallcaps("unblock a user") + "\n" +
			"➭ <b>/blocklist</b> — " + smallcaps("show blocked list") + "</blockquote>",
		"thumb": "<blockquote><b>⊚ " + smallcaps("thumbnail") + "</b></blockquote>\n" +
			"<blockquote expandable>Ὂe " + smallcaps("player panel always sends the track cover with now playing info.") + "</blockquote>",
		"start": "<blockquote><b>⊚ " + smallcaps("basic commands") + "</b></blockquote>\n" +
			"<blockquote expandable>➭ <b>/start</b> — " + smallcaps("open this menu") + "\n" +
			"➭ <b>/help</b> — " + smallcaps("command index") + "\n" +
			"➭ <b>/ping</b> — " + smallcaps("latency") + "\n" +
			"➭ <b>/id</b> — " + smallcaps("chat and user ids") + "\n" +
			"➭ <b>/stats</b> — " + smallcaps("bot stats") + "</blockquote>",
		"autoplay": "<blockquote><b>⊚ " + smallcaps("autoplay") + "</b></blockquote>\n" +
			"<blockquote expandable>➭ <b>/autoplay</b> — " + smallcaps("toggle autoplay") + "\n" +
			"Ὂe " + smallcaps("when enabled, related tracks keep joining the queue.") + "\n" +
			"➭ " + smallcaps("also toggle from the player panel button") + "</blockquote>",
		"playlist": "<blockquote><b>⊚ " + smallcaps("playlist") + "</b></blockquote>\n" +
			"<blockquote expandable>Ὂe " + smallcaps("send a youtube playlist link with") + " <b>/play</b> " + smallcaps("or") + " <b>/vplay</b>.\n" +
			"➭ " + smallcaps("tracks are queued in order") + "</blockquote>",
		"vclogs": "<blockquote><b>⊚ " + smallcaps("vc logs") + "</b></blockquote>\n" +
			"<blockquote expandable>Ὂe " + smallcaps("playback status stays on the player panel.") + "\n" +
			"➭ " + smallcaps("use") + " <b>/queue</b> " + smallcaps("to see what is next") + "</blockquote>",
		"inline": "<blockquote><b>⊚ " + smallcaps("player buttons") + "</b></blockquote>\n" +
			"<blockquote expandable>➭ <b>▷</b> " + smallcaps("resume") + "\n" +
			"➭ <b>II</b> " + smallcaps("pause") + "\n" +
			"➭ <b>⟳</b> " + smallcaps("replay") + "\n" +
			"➭ <b>‣‣I</b> " + smallcaps("skip") + "\n" +
			"➭ <b>▢</b> " + smallcaps("stop") + "\n" +
			"➭ <b>-15s</b> / <b>+15s</b> " + smallcaps("seek") + "\n" +
			"▣ " + smallcaps("group admins only") + "</blockquote>",
	}
	if body, ok := pages[key]; ok {
		return body
	}
	return helpMainHTML()
}
