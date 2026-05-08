package telegram

// InlineButton represents a button in an inline keyboard.
type InlineButton struct {
	Label string
	Data  string // callback data, max 64 bytes
}

// Sender abstracts Telegram API calls for testability.
type Sender interface {
	SendMessage(chatID int64, text string) (messageID int, err error)
	SendMessageWithButtons(chatID int64, text string, buttons [][]InlineButton) (messageID int, err error)
	EditMessage(chatID int64, messageID int, text string) error
	DeleteMessage(chatID int64, messageID int) error
	AnswerCallback(callbackID string, text string) error
}
