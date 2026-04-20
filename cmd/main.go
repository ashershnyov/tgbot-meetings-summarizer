package main

import (
	"log"

	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/bot"
)

func main() {
	bot, err := bot.New()
	if err != nil {
		log.Fatalf("error creating bot instance: %w", err)
	}

	err = bot.Run()
	if err != nil {
		log.Fatalf("error starting bot: %w", err)
	}
}
