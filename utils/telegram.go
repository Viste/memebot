package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramSender отправляет сообщения в Telegram
type TelegramSender struct {
	bot *tgbotapi.BotAPI
}

// NewTelegramSender создает новый отправитель
func NewTelegramSender(bot *tgbotapi.BotAPI) *TelegramSender {
	return &TelegramSender{bot: bot}
}

// sendPhoto отправляет фото через прямой API вызов для обхода проблем с типами
func (ts *TelegramSender) sendPhoto(chatID, fileID, caption string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", ts.bot.Token)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"photo":   fileID,
		"caption": caption,
	}

	return ts.sendRequest(url, payload)
}

// sendVideo отправляет видео через прямой API вызов
func (ts *TelegramSender) sendVideo(chatID, fileID, caption string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendVideo", ts.bot.Token)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"video":   fileID,
		"caption": caption,
	}

	return ts.sendRequest(url, payload)
}

// sendMediaGroup отправляет медиа группу через прямой API вызов
func (ts *TelegramSender) sendMediaGroup(chatID string, media []interface{}) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMediaGroup", ts.bot.Token)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"media":   media,
	}

	return ts.sendRequest(url, payload)
}

// sendRequest отправляет HTTP запрос к Telegram API
func (ts *TelegramSender) sendRequest(url string, payload map[string]interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %s", string(body))
	}

	return nil
}

// SendToChannelSimple отправляет фото в канал (упрощенный метод)
func SendToChannelSimple(bot *tgbotapi.BotAPI, channelID string, photo tgbotapi.FileID, caption string) error {
	sender := NewTelegramSender(bot)
	return sender.sendPhoto(channelID, string(photo), caption)
}

// SendVideoToChannelSimple отправляет видео в канал (упрощенный метод)
func SendVideoToChannelSimple(bot *tgbotapi.BotAPI, channelID string, video tgbotapi.FileID, caption string) error {
	sender := NewTelegramSender(bot)
	return sender.sendVideo(channelID, string(video), caption)
}

// SendMediaGroupToChannelSimple отправляет медиа группу в канал (упрощенный метод)
func SendMediaGroupToChannelSimple(bot *tgbotapi.BotAPI, channelID string, media []interface{}) error {
	sender := NewTelegramSender(bot)

	apiMedia := make([]interface{}, len(media))
	for i, m := range media {
		if inputMedia, ok := m.(tgbotapi.InputMediaPhoto); ok {
			apiMedia[i] = map[string]interface{}{
				"type":  "photo",
				"media": inputMedia.Media,
			}
		} else {
			return fmt.Errorf("unsupported media type: %T", m)
		}
	}

	return sender.sendMediaGroup(channelID, apiMedia)
}
