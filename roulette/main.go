package main

import (
	"context"
	stdlog "log"
	"os"
	"os/signal"

	"roulettebot"
	"roulettebot/store"

	"github.com/BurntSushi/toml"
	"github.com/caarlos0/env"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/language"
)

func main() {
	cfg := roulettebot.Config{
		Store: store.NewMemStore(),
		Queue: store.NewMemQueue(),
	}

	if err := env.Parse(&cfg); err != nil {
		stdlog.Fatalf("cant parse config: %v", err)
	}

	logger := log.New()
	level, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatalf("falied to parse log level: %v", err)
	}
	logger.Level = level
	cfg.Logger = logger

	// TODO: parse options from env
	bundle := i18n.NewBundle(language.Russian)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	bundle.MustLoadMessageFile("i18n/ru.toml")

	cfg.Bundle = bundle

	bot, err := roulettebot.NewBot(cfg)
	if err != nil {
		logger.WithField("err", err).Fatal("cant create bot")
	}

	botInfo := bot.BotInfo()
	logger.WithFields(log.Fields{
		"bot_nick": botInfo.Nick,
		"bot_name": botInfo.FirstName,
	}).Info("starting bot")

	ctx, cancel := context.WithCancel(context.Background())
	stop := make(chan struct{})

	go func() {
		err = bot.StartPolling(ctx)
		if err != context.Canceled && err != context.DeadlineExceeded {
			logger.WithFields(log.Fields{
				"err": err,
			}).Error("error from StartPolling")
		}
		close(stop)
	}()

	sigStop := make(chan os.Signal, 1)
	signal.Notify(sigStop, os.Interrupt)
	select {
	case <-sigStop:
		logger.Info("stopping bot by signal")
		cancel()
		<-stop
		logger.Info("stopped bot by signal")
	case <-stop:
		logger.Info("stopped bot by unknown reason")
	}
}
