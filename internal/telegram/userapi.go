package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// UserAPIConfig configures the Telegram User API client.
type UserAPIConfig struct {
	APIId       int
	APIHash     string
	PhoneNumber string
	SessionPath string
}

// Dialog represents a Telegram chat.
type Dialog struct {
	ID          int64
	Title       string
	Type        string // "user", "group", "channel"
	UnreadCount int
}

// UserMessage represents a message from chat history.
type UserMessage struct {
	ID         int
	SenderName string
	Text       string
	Date       time.Time
}

// UserAPIClient wraps gotd/td for Telegram User API access.
type UserAPIClient struct {
	client *telegram.Client
	api    *tg.Client
	cfg    UserAPIConfig
	logger *slog.Logger
}

// NewUserAPIClient creates a User API client.
func NewUserAPIClient(cfg UserAPIConfig, logger *slog.Logger) *UserAPIClient {
	storage := &session.FileStorage{Path: cfg.SessionPath}

	client := telegram.NewClient(cfg.APIId, cfg.APIHash, telegram.Options{
		SessionStorage: storage,
	})

	return &UserAPIClient{
		client: client,
		cfg:    cfg,
		logger: logger,
	}
}

// Run starts the client and calls fn with the authenticated API.
// Blocks until fn returns or ctx is cancelled.
func (c *UserAPIClient) Run(ctx context.Context, fn func(ctx context.Context) error) error {
	return c.client.Run(ctx, func(ctx context.Context) error {
		c.api = c.client.API()

		// Check if already authenticated.
		status, err := c.client.Auth().Status(ctx)
		if err != nil {
			return fmt.Errorf("userapi: check auth: %w", err)
		}

		if !status.Authorized {
			c.logger.Info("userapi: authentication required", "phone", c.cfg.PhoneNumber)
			// Interactive auth — requires user to enter code.
			flow := auth.NewFlow(
				auth.Constant(c.cfg.PhoneNumber, "", auth.CodeAuthenticatorFunc(
					func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
						// In production, prompt user via Telegram bot or stdin.
						return "", fmt.Errorf("userapi: interactive auth not implemented — run setup first")
					},
				)),
				auth.SendCodeOptions{},
			)
			if err := flow.Run(ctx, c.client.Auth()); err != nil {
				return fmt.Errorf("userapi: auth flow: %w", err)
			}
		}

		c.logger.Info("userapi: authenticated")
		return fn(ctx)
	})
}

// GetDialogs returns the user's chat list.
func (c *UserAPIClient) GetDialogs(ctx context.Context, limit int) ([]Dialog, error) {
	if c.api == nil {
		return nil, fmt.Errorf("userapi: not connected")
	}

	result, err := c.api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		Limit:      limit,
		OffsetPeer: &tg.InputPeerEmpty{},
	})
	if err != nil {
		return nil, fmt.Errorf("userapi: get dialogs: %w", err)
	}

	var dialogs []Dialog
	switch v := result.(type) {
	case *tg.MessagesDialogs:
		dialogs = extractDialogs(v.Dialogs, v.Chats, v.Users)
	case *tg.MessagesDialogsSlice:
		dialogs = extractDialogs(v.Dialogs, v.Chats, v.Users)
	}

	return dialogs, nil
}

// GetHistory returns message history for a chat.
func (c *UserAPIClient) GetHistory(ctx context.Context, peer tg.InputPeerClass, limit int) ([]UserMessage, error) {
	if c.api == nil {
		return nil, fmt.Errorf("userapi: not connected")
	}

	result, err := c.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  peer,
		Limit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("userapi: get history: %w", err)
	}

	var messages []UserMessage
	switch v := result.(type) {
	case *tg.MessagesMessages:
		messages = extractMessages(v.Messages)
	case *tg.MessagesMessagesSlice:
		messages = extractMessages(v.Messages)
	case *tg.MessagesChannelMessages:
		messages = extractMessages(v.Messages)
	}

	return messages, nil
}

// SendMessage sends a message via User API (requires approval).
func (c *UserAPIClient) SendMessage(ctx context.Context, peer tg.InputPeerClass, text string) error {
	if c.api == nil {
		return fmt.Errorf("userapi: not connected")
	}

	_, err := c.api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
		Peer:    peer,
		Message: text,
	})
	if err != nil {
		return fmt.Errorf("userapi: send message: %w", err)
	}

	return nil
}

func extractDialogs(dialogs []tg.DialogClass, chats []tg.ChatClass, users []tg.UserClass) []Dialog {
	chatMap := make(map[int64]string)
	for _, chat := range chats {
		switch c := chat.(type) {
		case *tg.Chat:
			chatMap[c.ID] = c.Title
		case *tg.Channel:
			chatMap[c.ID] = c.Title
		}
	}

	userMap := make(map[int64]string)
	for _, user := range users {
		switch u := user.(type) {
		case *tg.User:
			name := u.FirstName
			if u.LastName != "" {
				name += " " + u.LastName
			}
			userMap[u.ID] = name
		}
	}

	var result []Dialog
	for _, d := range dialogs {
		dialog, ok := d.(*tg.Dialog)
		if !ok {
			continue
		}

		var dlg Dialog
		dlg.UnreadCount = dialog.UnreadCount

		switch peer := dialog.Peer.(type) {
		case *tg.PeerUser:
			dlg.ID = peer.UserID
			dlg.Title = userMap[peer.UserID]
			dlg.Type = "user"
		case *tg.PeerChat:
			dlg.ID = peer.ChatID
			dlg.Title = chatMap[peer.ChatID]
			dlg.Type = "group"
		case *tg.PeerChannel:
			dlg.ID = peer.ChannelID
			dlg.Title = chatMap[peer.ChannelID]
			dlg.Type = "channel"
		}

		result = append(result, dlg)
	}

	return result
}

func extractMessages(messages []tg.MessageClass) []UserMessage {
	var result []UserMessage
	for _, m := range messages {
		msg, ok := m.(*tg.Message)
		if !ok {
			continue
		}
		result = append(result, UserMessage{
			ID:   msg.ID,
			Text: msg.Message,
			Date: time.Unix(int64(msg.Date), 0),
		})
	}
	return result
}
