package main

import (
	"log"

	"github.com/amarnathcjd/gogram/telegram"
)

func registerCallbackFallback() {
	Bot.On("callback:*", func(cb *telegram.CallbackQuery) error {
		return handleCallbackQuery(cb)
	})
	log.Println("Loaded module: callbacks")
}
