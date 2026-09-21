package main

import "log"

func registerCallbackFallback() {
	// handleCallbackQuery is already registered in registerHandlers.
	// A second Bot.On(callback) made every menu edit run twice and
	// the second pass dropped the blockquote.
	log.Println("Loaded module: callbacks")
}
