package utils

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// MediaGroup для группировки медиа
type MediaGroup struct {
	Media   []interface{}
	Caption string
	ChatID  int64
}

// MediaGroupManager управляет медиа группами
type MediaGroupManager struct {
	groups map[string]*MediaGroup
	timers map[string]*time.Timer
	mu     sync.RWMutex
}

func NewMediaGroupManager() *MediaGroupManager {
	return &MediaGroupManager{
		groups: make(map[string]*MediaGroup),
		timers: make(map[string]*time.Timer),
	}
}

func (mgm *MediaGroupManager) AddToGroup(groupID string, media interface{}, caption string, chatID int64) {
	mgm.mu.Lock()
	defer mgm.mu.Unlock()

	if _, exists := mgm.groups[groupID]; !exists {
		mgm.groups[groupID] = &MediaGroup{
			Media:   make([]interface{}, 0),
			Caption: caption,
			ChatID:  chatID,
		}
	}

	mgm.groups[groupID].Media = append(mgm.groups[groupID].Media, media)

	if mgm.groups[groupID].Caption == "" {
		mgm.groups[groupID].Caption = caption
	}
}

func (mgm *MediaGroupManager) SetTimer(groupID string, duration time.Duration, callback func(string)) {
	mgm.mu.Lock()
	defer mgm.mu.Unlock()

	if timer, exists := mgm.timers[groupID]; exists {
		timer.Stop()
	}

	mgm.timers[groupID] = time.AfterFunc(duration, func() {
		callback(groupID)
	})
}

func (mgm *MediaGroupManager) GetGroup(groupID string) (*MediaGroup, bool) {
	mgm.mu.RLock()
	defer mgm.mu.RUnlock()

	group, exists := mgm.groups[groupID]
	return group, exists
}

func (mgm *MediaGroupManager) RemoveGroup(groupID string) {
	mgm.mu.Lock()
	defer mgm.mu.Unlock()

	if timer, exists := mgm.timers[groupID]; exists {
		timer.Stop()
		delete(mgm.timers, groupID)
	}

	delete(mgm.groups, groupID)
}

// SplitMessage разбивает длинное сообщение на части
func SplitMessage(text string, maxLength int) []string {
	if len(text) <= maxLength {
		return []string{text}
	}

	var chunks []string
	for len(text) > maxLength {
		splitIndex := maxLength
		for i := maxLength - 1; i >= 0; i-- {
			if text[i] == ' ' || text[i] == '\n' {
				splitIndex = i
				break
			}
		}

		chunks = append(chunks, text[:splitIndex])
		text = text[splitIndex:]

		text = strings.TrimLeft(text, " \n")
	}

	if len(text) > 0 {
		chunks = append(chunks, text)
	}

	return chunks
}

// GetImageURL формирует URL для получения изображения
func GetImageURL(token, filePath string) string {
	return fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", token, filePath)
}

// GetSenderName извлекает имя отправителя из сообщения
func GetSenderName(message *tgbotapi.Message) (string, string) {
	var firstName, lastName string

	if message.From.UserName != "" {
		firstName = message.From.UserName
	} else if message.From.FirstName != "" && message.From.FirstName != "\xad" {
		firstName = message.From.FirstName
	} else {
		firstName = "Аноним"
	}

	if message.From.LastName != "" {
		lastName = message.From.LastName
	} else {
		lastName = ""
	}

	return firstName, lastName
}

// IsValidChat проверяет, разрешен ли чат для работы бота
func IsValidChat(chatTitle string) bool {
	return strings.Contains(chatTitle, "Подписчик Сталина Chat")
}

// SendReply отправляет ответ на сообщение с обработкой ошибок
func SendReply(bot *tgbotapi.BotAPI, message *tgbotapi.Message, text string) (*tgbotapi.Message, error) {
	chunks := SplitMessage(text, 4096)

	var lastMessage tgbotapi.Message
	var err error

	for i, chunk := range chunks {
		msg := tgbotapi.NewMessage(message.Chat.ID, chunk)
		if i == 0 {
			msg.ReplyToMessageID = message.MessageID
		}
		msg.ParseMode = tgbotapi.ModeHTML

		lastMessage, err = bot.Send(msg)
		if err != nil {
			msg.ParseMode = ""
			lastMessage, err = bot.Send(msg)
			if err != nil {
				return nil, err
			}
		}
	}

	return &lastMessage, nil
}

// SendToChannel отправляет сообщение в канал
func SendToChannel(bot *tgbotapi.BotAPI, channelID string, photo tgbotapi.FileID, caption string) error {
	if chatID, err := strconv.ParseInt(channelID, 10, 64); err == nil {
		msg := tgbotapi.NewPhoto(chatID, photo)
		msg.Caption = caption
		if _, err := bot.Send(msg); err == nil {
			return nil
		}
	}

	return SendToChannelSimple(bot, channelID, photo, caption)
}

// SendVideoToChannel отправляет видео в канал
func SendVideoToChannel(bot *tgbotapi.BotAPI, channelID string, video tgbotapi.FileID, caption string) error {
	if chatID, err := strconv.ParseInt(channelID, 10, 64); err == nil {
		msg := tgbotapi.NewVideo(chatID, video)
		msg.Caption = caption
		if _, err := bot.Send(msg); err == nil {
			return nil
		}
	}

	return SendVideoToChannelSimple(bot, channelID, video, caption)
}

// SendMediaGroupToChannel отправляет медиа группу в канал
func SendMediaGroupToChannel(bot *tgbotapi.BotAPI, channelID string, media []interface{}) error {
	// Сначала пытаемся стандартным способом для int64
	if chatID, err := strconv.ParseInt(channelID, 10, 64); err == nil {
		msg := tgbotapi.NewMediaGroup(chatID, media)
		if _, err := bot.Send(msg); err == nil {
			return nil
		}
	}

	return SendMediaGroupToChannelSimple(bot, channelID, media)
}
