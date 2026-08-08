package services

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"memebot/database"
	"memebot/metrics"
	"memebot/models"
	"memebot/utils"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// AutoPoster несколько раз в день постит в канал сгенерированный мем
// по мотивам одного из сегодняшних мемов чата.
type AutoPoster struct {
	bot         *tgbotapi.BotAPI
	openai      *OpenAIService
	channel     string
	postsPerDay float64
}

func NewAutoPoster(bot *tgbotapi.BotAPI, openai *OpenAIService, channel string, postsPerDay float64) *AutoPoster {
	return &AutoPoster{
		bot:         bot,
		openai:      openai,
		channel:     channel,
		postsPerDay: postsPerDay,
	}
}

// Start запускает цикл автопостинга в фоне. Отменяется через ctx.
func (ap *AutoPoster) Start(ctx context.Context) {
	if ap.postsPerDay <= 0 || ap.channel == "" {
		log.Printf("Channel meme autoposting disabled")
		return
	}

	log.Printf("Channel meme autoposting enabled: ~%.1f posts/day to %s", ap.postsPerDay, ap.channel)

	go func() {
		for {
			delay := ap.nextDelay()
			log.Printf("Next channel meme in %s", delay.Round(time.Minute))

			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}

			if err := ap.postGeneratedMeme(ctx); err != nil {
				log.Printf("Error posting generated meme to channel: %v", err)
			}
		}
	}()
}

// nextDelay — средний интервал 24ч/postsPerDay с джиттером ±50%,
// чтобы посты не выходили по расписанию как по будильнику.
func (ap *AutoPoster) nextDelay() time.Duration {
	base := time.Duration(float64(24*time.Hour) / ap.postsPerDay)
	jitter := 0.5 + rand.Float64()
	return time.Duration(float64(base) * jitter)
}

func (ap *AutoPoster) postGeneratedMeme(ctx context.Context) error {
	sourceURLs, memeContext := ap.todayMemeInspiration()

	var request string
	if memeContext != "" || len(sourceURLs) > 0 {
		request = "Ты ведёшь свой мемный канал. Выше — один из сегодняшних мемов подписчиков и твой комментарий к нему. Придумай СВОЙ мем по мотивам этой темы — не копию, а развитие или подкол — для поста в канал."
	} else {
		request = "Ты ведёшь свой мемный канал. Свежих мемов от подписчиков сегодня не было. Придумай самостоятельный смешной мем для поста в канал — про мемную культуру, будни канала или вечное."
	}

	imageData, caption, err := ap.openai.GenerateMemeRemake(ctx, sourceURLs, memeContext, request, "channel")
	if err != nil {
		return fmt.Errorf("generate channel meme: %w", err)
	}

	if err := utils.SendPhotoBytesToChannel(ap.bot, ap.channel, imageData, caption); err != nil {
		metrics.TrackChannelPostError("generated_meme_send_failed")
		return fmt.Errorf("send channel meme: %w", err)
	}

	metrics.TrackMemePosted("generated")
	log.Printf("Posted generated meme to channel: %s", caption)
	return nil
}

// todayMemeInspiration выбирает случайный сегодняшний мем из истории взаимодействий:
// возвращает его картинки и текстовый контекст (комментарии бота и обсуждение).
func (ap *AutoPoster) todayMemeInspiration() (urls []string, memeContext string) {
	dayStart := time.Now().Truncate(24 * time.Hour)

	var memeID string
	err := database.DB.Model(&models.MemeInteraction{}).
		Select("meme_id").
		Where("created_at >= ? AND role = ? AND (content LIKE ? OR content LIKE ?)",
			dayStart, "user", "[MEME_IMAGE:%", "[MEME_GROUP:%").
		Order("RANDOM()").
		Limit(1).
		Scan(&memeID).Error
	if err != nil || memeID == "" {
		if err != nil {
			log.Printf("todayMemeInspiration query error: %v", err)
		}
		return nil, ""
	}

	var interactions []models.MemeInteraction
	err = database.DB.
		Where("meme_id = ?", memeID).
		Order("created_at ASC").
		Limit(20).
		Find(&interactions).Error
	if err != nil {
		log.Printf("todayMemeInspiration history error: %v", err)
		return nil, ""
	}

	var b strings.Builder
	for _, entry := range interactions {
		content := entry.Content
		switch {
		case strings.HasPrefix(content, "[MEME_IMAGE: "):
			urls = append(urls, strings.TrimSuffix(strings.TrimPrefix(content, "[MEME_IMAGE: "), "]"))
		case strings.HasPrefix(content, "[MEME_GROUP: "):
			list := strings.TrimSuffix(strings.TrimPrefix(content, "[MEME_GROUP: "), "]")
			urls = append(urls, strings.Split(list, ", ")...)
		default:
			role := "Подписчик"
			if entry.Role == "assistant" {
				role = "Твой комментарий"
			}
			b.WriteString(role + ": " + content + "\n")
		}
	}

	return urls, b.String()
}
