package main

import (
	"fmt"
	"log"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/nikhil390u8o/GOMUSICV2/ntgcalls"
)

type callAPI struct {
	*ntgcalls.Client
}

func (c *callAPI) Close() {
	if c != nil && c.Client != nil {
		c.Free()
	}
}

func (c *callAPI) SeekBy(chatID int64, ms int64) error {
	path := currentPath(chatID)
	if path == "" {
		if s := peekCurrent(chatID); s != nil {
			path = s.FilePath
		}
	}
	if path == "" {
		return nil
	}
	video := false
	if s := peekCurrent(chatID); s != nil {
		video = s.Video
	}
	return c.SetStreamSources(chatID, ntgcalls.CaptureStream, buildMedia(path, video, getSeekState(chatID)))
}

var (
	Bot               *telegram.Client
	Assistant         *telegram.Client
	Calls             *callAPI
	botStartTime      = time.Now()
	assistantUsername string
	assistantID       int64
	activeCalls       = map[int64]telegram.InputGroupCall{}
)

func waitReady(c *telegram.Client, label string) error {
	if c == nil {
		return fmt.Errorf("%s client is nil", label)
	}
	var last error
	for i := 1; i <= 8; i++ {
		_ = c.Connect()
		me, err := c.GetMe()
		if err == nil && me != nil {
			log.Printf("%s ready @%s id=%d\n", label, me.Username, me.ID)
			return nil
		}
		last = err
		log.Printf("%s GetMe retry %d/8: %v\n", label, i, err)
		time.Sleep(time.Duration(i) * time.Second)
	}
	return last
}

func initClients() error {
	if BotToken == "" {
		return fmt.Errorf("BOT_TOKEN is empty - set it in .env")
	}
	if StringSession == "" {
		return fmt.Errorf("STRING_SESSION is empty - set it in .env")
	}

	botClient, err := telegram.NewClient(telegram.ClientConfig{
		AppID:         int32(APIID),
		AppHash:       APIHash,
		MemorySession: true,
		ParseMode:     "HTML",
		NoPreconnect:  true,
	})
	if err != nil {
		return err
	}
	if err := botClient.Connect(); err != nil {
		return fmt.Errorf("bot connect: %w", err)
	}
	if err := botClient.LoginBot(BotToken); err != nil {
		return fmt.Errorf("bot login: %w", err)
	}
	time.Sleep(1500 * time.Millisecond)
	_ = botClient.Connect()
	Bot = botClient

	encoded, err := decodePyrogramSessionString(StringSession)
	if err != nil {
		return err
	}
	asst, err := telegram.NewClient(telegram.ClientConfig{
		AppID:         int32(APIID),
		AppHash:       APIHash,
		StringSession: encoded,
		MemorySession: true,
		ParseMode:     "HTML",
		NoPreconnect:  true,
	})
	if err != nil {
		return err
	}
	if err := asst.Connect(); err != nil {
		return fmt.Errorf("assistant connect: %w", err)
	}
	time.Sleep(800 * time.Millisecond)
	_ = asst.Connect()
	Assistant = asst

	Calls = &callAPI{Client: ntgcalls.NTgCalls()}
	Calls.OnStreamEnd(func(chat int64, t ntgcalls.StreamType, d ntgcalls.StreamDevice) {
		go handleStreamEnd(chat)
	})
	Calls.OnConnectionChange(func(chat int64, info ntgcalls.NetworkInfo) {
		connectMu.Lock()
		ch := connectWait[chat]
		connectMu.Unlock()
		if ch == nil {
			return
		}
		switch info.State {
		case ntgcalls.Connected:
			select {
			case ch <- nil:
			default:
			}
		case ntgcalls.Failed, ntgcalls.Timeout, ntgcalls.Closed:
			select {
			case ch <- fmt.Errorf("ntgcalls state %v", info.State):
			default:
			}
		}
	})
	return nil
}

func setBotCommands() {
	cmds := []*telegram.BotCommand{
		{Command: "start", Description: "start the bot"},
		{Command: "help", Description: "get help menu"},
		{Command: "play", Description: "play a song"},
		{Command: "pause", Description: "pause playback"},
		{Command: "resume", Description: "resume playback"},
		{Command: "skip", Description: "skip song"},
		{Command: "stop", Description: "stop and clear"},
		{Command: "ping", Description: "bot stats"},
	}
	_, err := Bot.BotsSetBotCommands(&telegram.BotCommandScopeDefault{}, "", cmds)
	if err != nil {
		log.Println("Could not set bot commands:", err)
	}
}
