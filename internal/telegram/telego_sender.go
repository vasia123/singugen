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

func (s *TelegoSender) SendMessageWithButtons(chatID int64, text string, buttons [][]InlineButton) (int, error) {
	var rows [][]telego.InlineKeyboardButton
	for _, row := range buttons {
		var tgRow []telego.InlineKeyboardButton
		for _, btn := range row {
			tgRow = append(tgRow, telego.InlineKeyboardButton{
				Text:         btn.Label,
				CallbackData: btn.Data,
			})
		}
		rows = append(rows, tgRow)
	}

	params := &telego.SendMessageParams{
		ChatID: tu.ID(chatID),
		Text:   text,
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: rows,
		},
	}

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

func (s *TelegoSender) AnswerCallback(callbackID string, text string) error {
	return s.bot.AnswerCallbackQuery(s.ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
	})
}
