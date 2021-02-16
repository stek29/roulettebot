package roulettebot

import (
	"context"
	"fmt"

	botgolang "github.com/mail-ru-im/bot-golang"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/language"
)

type Bot struct {
	bot       *botgolang.Bot
	cfg       Config
	localizer *i18n.Localizer
	logger    *log.Logger
}

func NewBot(cfg Config) (*Bot, error) {
	bot := Bot{
		cfg: cfg,
	}

	if cfg.Logger != nil {
		bot.logger = cfg.Logger
	} else {
		bot.logger = log.StandardLogger()
	}

	var err error

	var opts []botgolang.BotOption
	if cfg.APIURL != "" {
		opts = append(opts, botgolang.BotApiURL(cfg.APIURL))
	}

	bot.bot, err = botgolang.NewBot(cfg.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("cant create botgolang bot: %w", err)
	}

	// TODO: localizer per user
	bot.localizer = i18n.NewLocalizer(cfg.Bundle, language.Russian.String())

	return &bot, nil
}

func (b *Bot) BotInfo() *botgolang.BotInfo {
	return b.bot.Info
}

func (b *Bot) StartPolling(ctx context.Context) error {
	events := b.bot.GetUpdatesChannel(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e := <-events:
			// TODO: workers with limited goroutine count on highload
			go func() {
				defer func() {
					if r := recover(); r != nil {
						b.logger.WithFields(log.Fields{
							"error":    r,
							"event_id": e.EventID,
						}).Error("panic during event handling")
					}
				}()

				b.handleApiEvent(ctx, &e)
			}()
		}
	}
}
