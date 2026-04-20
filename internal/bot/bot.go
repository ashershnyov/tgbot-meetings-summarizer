package bot

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/bot/client/giga"
	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/bot/client/salute"
	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/bot/config"
	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/bot/handler"
	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/bot/storage"
	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/db"
	"github.com/pressly/goose"
	tele "gopkg.in/telebot.v3"
)

// Bot defines a new bot.
type Bot struct {
	bot  *tele.Bot
	db   db.DB
	giga giga.Client
}

// New returns a new bot instance.
func New() (*Bot, error) {
	cfg, err := config.New()
	if err != nil {
		return nil, fmt.Errorf("error creating bot instance: %w")
	}

	teleBot, err := tele.NewBot(tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		return nil, err
	}

	bot := Bot{bot: teleBot}

	bot.db, err = db.NewPostgres(context.Background(), cfg.DBAddress)
	if err != nil {
		return nil, fmt.Errorf("an error occured when opening DB: %w", err)
	}

	stg := storage.NewDB(bot.db)

	giga, err := giga.NewClient(cfg.GigaToken)
	if err != nil {
		return nil, fmt.Errorf("error creating gigachat client: %w", err)
	}

	saluteClient, err := salute.NewClient(cfg.SaluteToken)
	if err != nil {
		return nil, fmt.Errorf("error creating SaluteSpeech client: %w", err)
	}

	botHandler := handler.New(stg, giga, saluteClient)
	botHandler.Register(bot.bot)

	return &bot, nil
}

// Run starts the bot.
func (b *Bot) Run() error {
	err := goose.Up(b.db.SQLDB(), "./migrations")
	if err != nil {
		return fmt.Errorf("an error occurred when starting bot: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		b.bot.Start()
	}()

	for range ctx.Done() {
		b.bot.Stop()
		b.giga.Stop()
	}

	return nil
}
