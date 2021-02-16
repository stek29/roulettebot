package roulettebot

import (
	botgolang "github.com/mail-ru-im/bot-golang"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
)

// l is shortcut for localization
func (b *Bot) l(key string, data map[string]interface{}) string {
	// TODO: localizer per user
	return b.localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: data,
	})
}

// sendLog sends message, and logs error if there is any
func (b *Bot) sendLog(eID int, m *botgolang.Message) error {
	err := m.Send()
	if err != nil {
		b.logger.WithFields(log.Fields{
			"event_id": eID,
			"error":    err,
		}).Error("cant send message")
	}
	return err
}
