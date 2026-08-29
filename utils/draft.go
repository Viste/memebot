package utils

import (
	"log"
	"math/rand"
	"sync"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

type Draft struct {
	bot      *tgbotapi.BotAPI
	chatID   int64
	threadID int
	draftID  int

	mu       sync.Mutex
	last     time.Time
	disabled bool
}

const draftMinInterval = 600 * time.Millisecond

func NewDraft(bot *tgbotapi.BotAPI, message *tgbotapi.Message) *Draft {
	return &Draft{
		bot:      bot,
		chatID:   message.Chat.ID,
		threadID: message.MessageThreadID,
		draftID:  rand.Intn(1<<30) + 1,
	}
}

func (d *Draft) Thinking() {
	d.send(tgbotapi.SendMessageDraftConfig{
		ChatConfig:          tgbotapi.ChatConfig{ChatID: d.chatID},
		MessageThreadID:     d.threadID,
		DraftID:             d.draftID,
		ThinkingPlaceholder: true,
	})
}

func (d *Draft) Update(partial string) {
	if partial == "" {
		return
	}

	d.mu.Lock()
	if d.disabled || time.Since(d.last) < draftMinInterval {
		d.mu.Unlock()
		return
	}
	d.last = time.Now()
	d.mu.Unlock()

	d.send(tgbotapi.SendMessageDraftConfig{
		ChatConfig:      tgbotapi.ChatConfig{ChatID: d.chatID},
		MessageThreadID: d.threadID,
		DraftID:         d.draftID,
		Text:            StripMarkdown(partial),
	})
}

func (d *Draft) send(cfg tgbotapi.SendMessageDraftConfig) {
	d.mu.Lock()
	if d.disabled {
		d.mu.Unlock()
		return
	}
	d.mu.Unlock()

	if _, err := d.bot.Request(cfg); err != nil {
		d.mu.Lock()
		d.disabled = true
		d.mu.Unlock()
		log.Printf("Draft streaming disabled for chat %d: %v", d.chatID, err)
	}
}

func KeepChatAction(bot *tgbotapi.BotAPI, message *tgbotapi.Message, action string) (stop func()) {
	done := make(chan struct{})
	var once sync.Once

	send := func() {
		cfg := tgbotapi.NewChatAction(message.Chat.ID, action)
		cfg.MessageThreadID = message.MessageThreadID
		if _, err := bot.Request(cfg); err != nil {
			log.Printf("Chat action %s error: %v", action, err)
		}
	}

	go func() {
		send()
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				send()
			}
		}
	}()

	return func() { once.Do(func() { close(done) }) }
}
