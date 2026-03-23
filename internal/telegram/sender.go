package telegram

// Sender abstracts Telegram API calls for testability.
type Sender interface {
	SendMessage(chatID int64, text string) (messageID int, err error)
	EditMessage(chatID int64, messageID int, text string) error
	DeleteMessage(chatID int64, messageID int) error
}
