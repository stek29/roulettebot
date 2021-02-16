package roulettebot

import (
	"context"
	"fmt"
	"regexp"
	"roulettebot/store"
	"strings"

	botgolang "github.com/mail-ru-im/bot-golang"
	log "github.com/sirupsen/logrus"
)

var (
	botCmdRegexp = regexp.MustCompile("^/[a-zA-Z0-9_]+")
)

func (b *Bot) handleApiEvent(ctx context.Context, e *botgolang.Event) {
	b.logger.WithFields(log.Fields{
		"event_id":   e.EventID,
		"event_type": e.Type,
	}).Debug("handling event")

	switch e.Type {
	case botgolang.NEW_MESSAGE:
		msg := e.Payload.Message()

		cmd := botCmdRegexp.FindString(msg.Text)
		if cmd != "" {
			cmd = strings.ToLower(cmd[1:])
			b.handleCommand(e, cmd)
			break
		}

		b.handleNewMessage(e)
	case botgolang.EDITED_MESSAGE:
		msg := b.bot.NewTextMessage(e.Payload.Chat.ID, b.l("EditNotSupported", nil))
		b.sendLog(e.EventID, msg)

	case botgolang.DELETED_MESSAGE:
		msg := b.bot.NewTextMessage(e.Payload.Chat.ID, b.l("DeleteNotSupported", nil))
		b.sendLog(e.EventID, msg)
	}
}

func (b *Bot) handleCommand(e *botgolang.Event, cmd string) {
	b.logger.WithFields(log.Fields{
		"event_id": e.EventID,
		"command":  cmd,
	}).Debug("handling command")

	msg := e.Payload.Message()

	switch cmd {
	case "start", "help":
		rmsg := b.bot.NewTextMessage(msg.Chat.ID, b.l("HelpMessage", map[string]interface{}{
			"Name": e.Payload.From.FirstName,
		}))
		b.sendLog(e.EventID, rmsg)
	case "debug":
		rmsg := b.bot.NewTextMessage(msg.Chat.ID, fmt.Sprintf("```\n%+v\n```", e.Payload))
		b.sendLog(e.EventID, rmsg)

	case "newchat":
		userID := e.Payload.Chat.ID

		if b.cfg.Store.HasPair(userID) {
			rmsg := b.bot.NewTextMessage(userID, b.l("NewChat_AlreadyInChat", nil))
			b.sendLog(e.EventID, rmsg)
			break
		}

		// comment out to enable selfchats for debugging
		if b.cfg.Queue.Has(userID) {
			rmsg := b.bot.NewTextMessage(msg.Chat.ID, b.l("NewChat_AlreadyInQueue", nil))
			b.sendLog(e.EventID, rmsg)
			break
		}

		queued, err := b.queueNewChat(e.EventID, userID)
		if err == store.ErrUserExists {
			rmsg := b.bot.NewTextMessage(userID, b.l("NewChat_AlreadyInQueue", nil))
			b.sendLog(e.EventID, rmsg)
			break
		}

		if err != nil {
			rmsg := b.bot.NewTextMessage(userID, b.l("UnknownError", map[string]interface{}{
				"Error": err,
			}))
			b.sendLog(e.EventID, rmsg)
			break
		}

		if queued {
			rmsg := b.bot.NewTextMessage(userID, b.l("NewChat_AddedToQueue", nil))
			b.sendLog(e.EventID, rmsg)
		}

	case "stopchat":
		userID := e.Payload.Chat.ID
		eID := e.EventID

		if b.cfg.Store.HasPair(userID) {
			err := b.breakChat(e.EventID, userID)
			if err != nil && err != store.ErrUserNotFound {
				b.logger.WithFields(log.Fields{
					"event_id": eID,
					"user_id":  userID,
					"error":    err,
				}).Error("cant break chat")
			}
			break
		}

		if b.cfg.Queue.Has(userID) {
			err := b.removeFromQueue(e.EventID, userID)
			if err != nil && err != store.ErrUserNotFound {
				b.logger.WithFields(log.Fields{
					"event_id": eID,
					"user_id":  userID,
					"error":    err,
				}).Error("cant remove from queue")
			}
			break
		}

		rmsg := b.bot.NewTextMessage(userID, b.l("StopChat_NotInChat", nil))
		b.sendLog(e.EventID, rmsg)

	default:
		msg.Text = b.l("UnknownCommand", map[string]interface{}{
			"Command": cmd,
		})
		b.sendLog(e.EventID, msg)
	}
}

func (b *Bot) handleNewMessage(e *botgolang.Event) {
	eID := e.EventID
	userID := e.Payload.Chat.ID

	pairID, err := b.cfg.Store.GetPair(userID)
	if err != nil {
		if err != store.ErrUserNotFound {
			b.logger.WithFields(log.Fields{
				"event_id": eID,
				"user_id":  userID,
				"error":    err,
			}).Error("cant get pair")
			return
		}

		if b.cfg.Queue.Has(userID) {
			rmsg := b.bot.NewTextMessage(userID, b.l("NewMessage_InQueue", nil))
			b.sendLog(e.EventID, rmsg)
		} else {
			rmsg := b.bot.NewTextMessage(userID, b.l("NewMessage_NotInChat", nil))
			b.sendLog(e.EventID, rmsg)
		}

		return
	}

	msg := e.Payload.Message()
	hadUnsupportedParts := false

	for _, p := range e.Payload.Parts {
		switch p.Type {
		case botgolang.FORWARD, botgolang.REPLY:
			if !hadUnsupportedParts {
				// if first part, send warning to pairID
				rmsg := b.bot.NewTextMessage(pairID, b.l("MessageHadInvalidParts", nil))
				b.sendLog(e.EventID, rmsg)
				hadUnsupportedParts = true
			}

			pmsg := b.bot.NewMessageFromPart(p.Payload.PartMessage)
			pmsg.Chat.ID = pairID
			b.sendLog(e.EventID, pmsg)
		}
	}

	if hadUnsupportedParts {
		rmsg := b.bot.NewTextMessage(userID, b.l("MessageHadInvalidParts", nil))
		b.sendLog(e.EventID, rmsg)
	}

	msg.Chat.ID = pairID
	b.sendLog(e.EventID, msg)

	b.logger.WithFields(log.Fields{
		"event_id":              eID,
		"user_id":               userID,
		"pair_id":               pairID,
		"had_unsupported_parts": hadUnsupportedParts,
	}).Debug("message relayed")
}

// queued is true if chat was queued and false if chat was started immideately
func (b *Bot) queueNewChat(eID int, userID string) (bool, error) {
	b.logger.WithFields(log.Fields{
		"event_id": eID,
		"user_id":  userID,
	}).Debug("trying to queue chat")

	pairID, err := b.cfg.Queue.Pick()
	if err == nil {
		return false, b.establishChat(eID, userID, pairID)
	}
	if err != store.ErrQueueEmpty {
		return false, err
	}

	err = b.cfg.Queue.Add(userID)
	if err != nil {
		return true, err
	}

	return true, nil
}

func (b *Bot) establishChat(eID int, userID, pairID string) error {
	err := b.cfg.Store.SetPair(userID, pairID)
	if err != nil {
		return fmt.Errorf("failed to set pair: %w", err)
	}

	b.logger.WithFields(log.Fields{
		"event_id": eID,
		"user_id":  userID,
		"pair_id":  pairID,
	}).Info("chat established")

	rmsg := b.bot.NewTextMessage(userID, b.l("NewPairFound", nil))
	rmsg.Chat.ID = userID
	b.sendLog(eID, rmsg)
	rmsg.Chat.ID = pairID
	b.sendLog(eID, rmsg)

	return nil
}

func (b *Bot) breakChat(eID int, userID string) error {
	pairID, err := b.cfg.Store.PopPair(userID)
	if err != nil {
		return err
	}

	b.logger.WithFields(log.Fields{
		"event_id": eID,
		"user_id":  userID,
		"pair_id":  pairID,
	}).Info("chat stopped")

	rmsg := b.bot.NewTextMessage(userID, b.l("StopChat_YouHaveLeft", nil))
	b.sendLog(eID, rmsg)

	rmsg = b.bot.NewTextMessage(pairID, b.l("StopChat_PairHasLeft", nil))
	b.sendLog(eID, rmsg)

	return nil
}

func (b *Bot) removeFromQueue(eID int, userID string) error {
	err := b.cfg.Queue.Remove(userID)
	if err != nil {
		return err
	}

	b.logger.WithFields(log.Fields{
		"event_id": eID,
		"user_id":  userID,
	}).Info("user removed from queue")

	rmsg := b.bot.NewTextMessage(userID, b.l("StopChat_RemovedFromQueue", nil))
	b.sendLog(eID, rmsg)

	return nil
}
