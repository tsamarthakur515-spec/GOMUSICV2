package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if err := loadConfig(); err != nil {
		log.Fatal(err)
	}
	if LoggerID == 0 {
		log.Println("LOGGER_ID / LOG_GROUP_ID is empty — start/play logs disabled")
	} else {
		log.Println("start/play logs will go to", LoggerID)
	}

	startStore()
	log.Println("Memory store ready (no MongoDB).")

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("GOMUSICV2 is running"))
		})
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("OK"))
		})
		addr := fmt.Sprintf(":%d", Port)
		log.Println("health server on port", Port)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Println("health server:", err)
		}
	}()

	if err := initClients(); err != nil {
		log.Fatal("client start failed:", err)
	}
	log.Println("Bot client started")

	me, err := retryGetMe(Bot, 5)
	if err != nil {
		log.Fatal("bot GetMe failed:", err)
	}
	log.Printf("Bot: @%s\n", me.Username)
	setBotCommands()

	if am, err := retryGetMe(Assistant, 5); err != nil {
		log.Println("Assistant GetMe failed (bot will still run):", err)
		assistantUsername = "sykrs"
	} else {
		assistantUsername = am.Username
		assistantID = am.ID
		log.Printf("Assistant: @%s id=%d\n", assistantUsername, assistantID)
	}

	registerHandlers()
	registerCallbackFallback()
	log.Println("draining old telegram updates for 8s")
	go func() {
		time.Sleep(8 * time.Second)
		enableUpdates()
		log.Println("GOMUSICV2 is ready for commands")
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("GOMUSICV2 stopped")
		if Bot != nil {
			Bot.Stop()
		}
		if Assistant != nil {
			Assistant.Stop()
		}
		if Calls != nil {
			Calls.Close()
		}
		os.Exit(0)
	}()

	Bot.Idle()
}

func retryGetMe(c *telegram.Client, tries int) (*telegram.UserObj, error) {
	var last error
	for i := 1; i <= tries; i++ {
		me, err := c.GetMe()
		if err == nil && me != nil {
			return me, nil
		}
		last = err
		log.Printf("GetMe retry %d/%d: %v\n", i, tries, err)
		time.Sleep(time.Duration(i) * 2 * time.Second)
	}
	return nil, last
}
