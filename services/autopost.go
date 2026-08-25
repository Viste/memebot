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
	sourceURLs, memeContext := ap.recentMemeInspiration()

	var request string
	if memeContext != "" || len(sourceURLs) > 0 {
		request = "Ты ведёшь свой мемный канал. Выше — один из недавних мемов подписчиков и твой комментарий к нему. Придумай СВОЙ мем по мотивам этой темы — не копию, а развитие или подкол — для поста в канал."
	} else {
		request = "Ты ведёшь свой мемный канал. Свежих мемов от подписчиков не было. Придумай самостоятельный смешной мем для поста в канал — про мемную культуру, будни канала или вечное."
	}

	imageData, caption, err := ap.openai.GenerateMemeRemake(ctx, sourceURLs, memeContext, request, "channel")
	if err != nil {
		return fmt.Errorf("generate channel meme: %w", err)
	}

	if err := utils.SendPhotoBytesToChannel(ap.bot, ap.channel, imageData, ""); err != nil {
		metrics.TrackChannelPostError("generated_meme_send_failed")
		return fmt.Errorf("send channel meme: %w", err)
	}

	metrics.TrackMemePosted("generated")
	log.Printf("Posted generated meme to channel (idea caption was: %s)", caption)
	return nil
}

// recentMemeInspiration выбирает случайный недавний мем: сперва за последние сутки,
// если пусто — за неделю. Кандидаты берутся из таблицы memes (всё, что бот запостил в канал
// из личек) и из meme_interactions (что бот комментировал в группе).
// Возвращает свежие ссылки на картинки и текстовый контекст.
func (ap *AutoPoster) recentMemeInspiration() (urls []string, memeContext string) {
	for _, window := range []time.Duration{24 * time.Hour, 7 * 24 * time.Hour} {
		since := time.Now().Add(-window)
		candidates := ap.candidateMemes(since)

		log.Printf("Autopost inspiration: %d candidate memes in last %s", len(candidates), window)

		if len(candidates) == 0 {
			continue
		}

		memeID := candidates[rand.Intn(len(candidates))]
		urls, memeContext = ap.memeDetails(memeID)
		log.Printf("Autopost inspiration: picked meme %s (%d images, context %d chars)", memeID, len(urls), len(memeContext))
		return urls, memeContext
	}

	return nil, ""
}

// candidateMemes собирает уникальные ID мемов из обоих источников.
func (ap *AutoPoster) candidateMemes(since time.Time) []string {
	seen := make(map[string]bool)
	var out []string
	add := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}

	for _, id := range RecentChannelPhotoIDs(since) {
		add(id)
	}

	var interactionIDs []string
	err := database.DB.Model(&models.MemeInteraction{}).
		Distinct().
		Where("created_at >= ? AND role = ? AND (content LIKE ? OR content LIKE ?)",
			since, "user", "[MEME_IMAGE:%", "[MEME_GROUP:%").
		Pluck("meme_id", &interactionIDs).Error
	if err != nil {
		log.Printf("candidateMemes interactions query error: %v", err)
	}
	for _, id := range interactionIDs {
		add(id)
	}

	return out
}

// memeDetails собирает по ID мема свежие ссылки на картинки и текстовый контекст.
// Для одиночных фото ID — это telegram file_id, по нему берём свежую ссылку через getFile
// (старые ссылки из истории живут ~час и обычно уже протухли).
func (ap *AutoPoster) memeDetails(memeID string) (urls []string, memeContext string) {
	var b strings.Builder

	if !strings.HasPrefix(memeID, "group_") {
		file, err := ap.bot.GetFile(tgbotapi.FileConfig{FileID: memeID})
		if err != nil {
			log.Printf("memeDetails getFile %s error: %v", memeID, err)
		} else if file.FilePath != "" {
			urls = append(urls, utils.GetImageURL(ap.bot.Token, file.FilePath))
		}

		var meme models.Meme
		if err := database.DB.Where("file_id = ?", memeID).First(&meme).Error; err == nil {
			name := "аноним"
			if meme.FirstName != nil && *meme.FirstName != "" {
				name = *meme.FirstName
				if meme.LastName != nil && *meme.LastName != "" {
					name += " " + *meme.LastName
				}
			}
			b.WriteString("Мем прислал в канал подписчик " + name + ".\n")
		}
	}

	var interactions []models.MemeInteraction
	err := database.DB.
		Where("meme_id = ?", memeID).
		Order("created_at ASC").
		Limit(20).
		Find(&interactions).Error
	if err != nil {
		log.Printf("memeDetails history error: %v", err)
	}

	for _, entry := range interactions {
		content := entry.Content
		switch {
		case strings.HasPrefix(content, "[MEME_IMAGE: "):
			if len(urls) == 0 {
				urls = append(urls, strings.TrimSuffix(strings.TrimPrefix(content, "[MEME_IMAGE: "), "]"))
			}
		case strings.HasPrefix(content, "[MEME_GROUP: "):
			if len(urls) == 0 {
				list := strings.TrimSuffix(strings.TrimPrefix(content, "[MEME_GROUP: "), "]")
				urls = append(urls, strings.Split(list, ", ")...)
			}
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
