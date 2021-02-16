package roulettebot

import (
	"roulettebot/store"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
)

// TODO: clean up, move config to app and keep only vars here, make options based
type Config struct {
	Token    string `env:"API_TOKEN,required"`
	APIURL   string `env:"API_URL"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	Queue store.Queue `env:"-"`
	Store store.Store `env:"-"`

	Bundle *i18n.Bundle
	Logger *log.Logger
}
