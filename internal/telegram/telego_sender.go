package telegram

import (
	"context"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// TelegoSender implements Sender using the telego library.
type TelegoSender struct {
	bot *telego.Bot
	ctx context.Context
}

// NewTelegoSender creates a Sender backed by a telego.Bot.
func NewTelegoSender(ctx context.Context, bot *telego.Bot) *TelegoSender {
	return &TelegoSender{bot: bot, ctx: ctx}
}

func (s *TelegoSender) SendMessage(chatID int64, text string) (int, error) {
	params := tu.Message(tu.ID(chatID), text)
	sent, err := s.bot.SendMessage(s.ctx, params)
	if err != nil {
		return 0, err
	}
	return sent.MessageID, nil
}

func (s *TelegoSender) EditMessage(chatID int64, messageID int, text string) error {
	_, err := s.bot.EditMessageText(s.ctx, &telego.EditMessageTextParams{
		ChatID:    tu.ID(chatID),
		MessageID: messageID,
		Text:      text,
	})
	return err
}

func (s *TelegoSender) DeleteMessage(chatID int64, messageID int) error {
	return s.bot.DeleteMessage(s.ctx, &telego.DeleteMessageParams{
		ChatID:    tu.ID(chatID),
		MessageID: messageID,
	})
}
