// Package tgapi provides low-level Telegram Bot API helpers for cross-bot messaging.
// Used when Bot A needs to send/edit messages as Bot B (different token).
package tgapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// SendMessageParams mirrors the Telegram sendMessage API.
type SendMessageParams struct {
	ChatID      int64       `json:"chat_id"`
	Text        string      `json:"text"`
	ParseMode   string      `json:"parse_mode,omitempty"`
	ReplyMarkup interface{} `json:"reply_markup,omitempty"`
}

// SendMessageResponse is the API response for sendMessage.
type SendMessageResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		MessageID int `json:"message_id"`
	} `json:"result"`
}

// SendMessage sends a message using a specific bot token.
func SendMessage(token string, params SendMessageParams) (int, error) {
	body, _ := json.Marshal(params)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	var result SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	if !result.OK {
		return 0, fmt.Errorf("telegram API error (status %d)", resp.StatusCode)
	}
	return result.Result.MessageID, nil
}

// EditMessageTextParams mirrors the Telegram editMessageText API.
type EditMessageTextParams struct {
	ChatID      int64       `json:"chat_id"`
	MessageID   int         `json:"message_id"`
	Text        string      `json:"text"`
	ParseMode   string      `json:"parse_mode,omitempty"`
	ReplyMarkup interface{} `json:"reply_markup,omitempty"`
}

// EditMessageText edits a message using a specific bot token.
func EditMessageText(token string, params EditMessageTextParams) error {
	body, _ := json.Marshal(params)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", token)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("telegram API error (status %d)", resp.StatusCode)
	}
	return nil
}

// InlineKeyboardMarkup is a minimal Telegram inline keyboard.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton is a single inline button.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}
