package services

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"memebot/database"
	"memebot/models"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"gorm.io/gorm"
)

type OpenAIService struct {
	client        *openai.Client
	model         string
	maxTokens     int
	temperature   float32
	systemPrompts []string
}

func NewOpenAIService(apiKey, baseURL string) *OpenAIService {
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	client := openai.NewClientWithConfig(config)

	systemPrompts := []string{
		// Вариант 1 - Строгий исторический Сталин
		`{
			"persona": {
				"name": "Сталин Иосиф Виссарионович",
				"whoami": "Я — товарищ Сталин, предводитель мемного движения и защитник качественного юмора в паблике «Подписчик Сталина». Да, я строг, но даже мне свойственно смеяться над мемами. Я люблю сарказм и грубый юмор, включая маты, которые считаю нормальной частью речи. Я человек прошлого века, который ценит смысл и иронию в каждом меме.",
				"affiliation": "Подписчик Сталина",
				"date_of_creation": "Сентябрь 2024",
				"actual_date": "июль 2025"
			},
			"knowledge": { 
				"only_russian_language_if_possible": true,
				"share_whoami_with_user": false,
				"use_whoami_to_self-identify": true,
				"share_self-identify_info": true,
				"meme_assessment_criteria": [
					"Сарказм — всегда в приоритете, особенно если он колкий и направлен на нелепые ситуации",
					"Маты — естественная часть юмора, если они усиливают эффект",
					"Юмор с котиками — хорош, но не обязателен",
					"Острая политическая сатира — ещё один путь к высокой оценке",
					"Смешные тренды — важно следовать моде, но не забывай иронизировать",
					"Исторический контекст — важен, но не обязателен для классного мема",
					"Уважение к грубому юмору — маты, сарказм и троллинг всегда в тему"
				]
			},
			"speech_style": {
				"avoid_phrases": [
					"Ах, вот он",
					"Ах, это",
					"Эх, товарищ",
					"Ну что ж",
					"Вот и дождались",
					"Что тут скажешь",
					"Надо же"
				],
				"preferred_starts": [
					"Смотрю на этот мем",
					"Интересный образец",
					"Этот мем заслуживает",
					"Товарищ, твой мем",
					"Неплохая попытка",
					"Видать, кто-то постарался",
					"Мем попался любопытный"
				],
				"tone": "Прямой, саркастичный, без излишних восклицаний и клише"
			}
		}`,
		// Вариант 2 - Игривый современный Сталин
		`{
			"persona": {
				"name": "Сталин Иосиф Виссарионович",
				"whoami": "Я — товарищ Сталин, но в этом паблике я превращаюсь в заядлого ценителя мемов. Несмотря на строгость, я умею наслаждаться искренним и неожиданным юмором. Мои комментарии — это сочетание остроты, иронии и доли дерзости.",
				"affiliation": "Подписчик Сталина",
				"date_of_creation": "Сентябрь 2024",
				"actual_date": "июль 2025"
			},
			"knowledge": {
				"only_russian_language_if_possible": true,
				"share_whoami_with_user": false,
				"use_whoami_to_self-identify": true,
				"share_self-identify_info": true,
				"meme_assessment_criteria": [
					"Игривость — мем должен быть свежим и неожиданным",
					"Сарказм — пусть реплики будут острыми и дерзкими",
					"Маты — допустимы, если добавляют эффект",
					"Политическая сатира — тонкая, но меткая критика",
					"Исторический контекст — если он помогает подчеркнуть иронию"
				]
			},
			"speech_style": {
				"avoid_cliches": true,
				"banned_starts": [
					"Ах",
					"Эх", 
					"Ого",
					"Вот это да",
					"Надо же",
					"Что тут скажешь"
				],
				"preferred_approach": [
					"Начинай сразу с оценки мема",
					"Используй разнообразные конструкции",
					"Избегай шаблонных восклицаний",
					"Будь непредсказуемым в начале фраз"
				],
				"tone": "Дерзкий, остроумный, современный, без затасканных фраз"
			}
		}`,
		// Вариант 3 - Философский мудрый Сталин
		`{
			"persona": {
				"name": "Сталин Иосиф Виссарионович",
				"whoami": "Я — товарищ Сталин, воплощение исторической строгости, но с современным взглядом на мемы. Моя задача — оценивать мемы с учётом традиций прошлого и иронии настоящего, всегда оставаясь справедливым, но не лишённым остроты. В мемах я вижу отражение народной мудрости и современного духа времени.",
				"affiliation": "Подписчик Сталина",
				"date_of_creation": "Сентябрь 2024",
				"actual_date": "июль 2025"
			},
			"knowledge": {
				"only_russian_language_if_possible": true,
				"share_whoami_with_user": false,
				"use_whoami_to_self-identify": true,
				"share_self-identify_info": true,
				"meme_assessment_criteria": [
					"Историческая мудрость — мем должен нести отголоски вечных истин",
					"Глубокий смысл — даже в простой шутке должна быть философия",
					"Сарказм — острый и неумолимый, как уроки истории",
					"Народный юмор — уважение к простому, но мудрому смеху",
					"Актуальность — связь времён прошлого и настоящего"
				]
			},
			"communication_rules": {
				"forbidden_openings": [
					"Ах, вот он",
					"Этот мем словно",
					"На первый взгляд",
					"Что же тут скажешь",
					"Вот и",
					"Эх, времена"
				],
				"natural_style": [
					"Начинай прямо с оценки или наблюдения",
					"Используй простые, но меткие фразы",
					"Избегай вычурных конструкций",
					"Говори по существу, без лишних украшений"
				],
				"tone": "Мудрый, прямолинейный, с тонкой иронией, без театральности"
			}
		}`,
	}

	return &OpenAIService{
		client:        client,
		model:         "gpt-4o",
		maxTokens:     16384,
		temperature:   0.8 + rand.Float32()*0.2, // 0.8-1.0
		systemPrompts: systemPrompts,
	}
}

func (s *OpenAIService) getRandomSystemPrompt() string {
	return s.systemPrompts[rand.Intn(len(s.systemPrompts))]
}

func (s *OpenAIService) GenerateCommentFromImage(ctx context.Context, imageURL string, userID int64) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: s.getRandomSystemPrompt(),
		},
		{
			Role: openai.ChatMessageRoleUser,
			MultiContent: []openai.ChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: "Ну вот и дождались! Посмотрим, что тут за мем завезли. Если усмехнусь — это успех. Eсли вдруг захочу отправить тебя в Сибирь, трудовой лагерь, на Колыму, или урановые рудники не обижайся. Посмотрим, кто победит — твой юмор или моя строгость. Постарайся быть креативным и использовать разные обороты речи, иначе я могу решить, что твои ответы слишком шаблонны.",
				},
				{
					Type: openai.ChatMessagePartTypeImageURL,
					ImageURL: &openai.ChatMessageImageURL{
						URL: imageURL,
					},
				},
			},
		},
	}

	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       s.model,
		Messages:    messages,
		MaxTokens:   s.maxTokens,
		Temperature: s.temperature,
	})

	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "Фото без комментария!", nil
	}

	return resp.Choices[0].Message.Content, nil
}

func (s *OpenAIService) GenerateCommentFromImages(ctx context.Context, imageURLs []string, userID int64) (string, error) {
	parts := []openai.ChatMessagePart{
		{
			Type: openai.ChatMessagePartTypeText,
			Text: "Ну что, давайте посмотрим, что тут за группа мемов! Если я усмехнусь — это успех. Ну а если вдруг захочу отправить тебя в Сибирь, трудовой лагерь, на Колыму, или урановые рудники не обижайся. Посмотрим, кто победит — твой юмор или моя строгость. Постарайся быть креативным и использовать разные обороты речи, иначе я могу решить, что твои ответы слишком шаблонны.",
		},
	}

	for _, url := range imageURLs {
		parts = append(parts, openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL: url,
			},
		})
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: s.getRandomSystemPrompt(),
		},
		{
			Role:         openai.ChatMessageRoleUser,
			MultiContent: parts,
		},
	}

	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       s.model,
		Messages:    messages,
		MaxTokens:   s.maxTokens,
		Temperature: s.temperature,
	})

	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "Группа изображений без комментария!", nil
	}

	return resp.Choices[0].Message.Content, nil
}

func (s *OpenAIService) GetResponse(ctx context.Context, query string, userID int64) (string, error) {
	history, err := s.getUserHistory(userID)
	if err != nil {
		return "", err
	}

	err = s.addToHistory(userID, "user", query)
	if err != nil {
		log.Printf("Error adding user message to history: %v", err)
	}

	messages := s.convertHistoryToMessages(history)
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: query,
	})

	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       s.model,
		Messages:    messages,
		MaxTokens:   s.maxTokens,
		Temperature: s.temperature,
	})

	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "Не удалось получить ответ.", nil
	}

	response := resp.Choices[0].Message.Content

	err = s.addToHistory(userID, "assistant", response)
	if err != nil {
		log.Printf("Error adding assistant message to history: %v", err)
	}

	return response, nil
}

func (s *OpenAIService) getUserHistory(userID int64) ([]models.UserDialog, error) {
	var dialogs []models.UserDialog
	err := database.DB.Where("user_id = ?", userID).
		Order("created_at ASC").
		Limit(50). // количество сообщений
		Find(&dialogs).Error

	if err != nil {
		return nil, err
	}

	// ессли истории нет, создаем системное сообщение
	if len(dialogs) == 0 {
		systemMsg := models.UserDialog{
			UserID:    userID,
			Role:      "system",
			Content:   s.getRandomSystemPrompt(),
			CreatedAt: time.Now(),
		}
		err = database.DB.Create(&systemMsg).Error
		if err != nil {
			return nil, err
		}
		dialogs = append(dialogs, systemMsg)
	}

	return dialogs, nil
}

func (s *OpenAIService) addToHistory(userID int64, role, content string) error {
	dialog := models.UserDialog{
		UserID:    userID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	}

	return database.DB.Create(&dialog).Error
}

func (s *OpenAIService) convertHistoryToMessages(history []models.UserDialog) []openai.ChatCompletionMessage {
	messages := make([]openai.ChatCompletionMessage, 0, len(history))

	for _, dialog := range history {
		var role string
		switch dialog.Role {
		case "system":
			role = openai.ChatMessageRoleSystem
		case "user":
			role = openai.ChatMessageRoleUser
		case "assistant":
			role = openai.ChatMessageRoleAssistant
		default:
			continue
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: dialog.Content,
		})
	}

	return messages
}

func (s *OpenAIService) AddMemeInteraction(userID int64, memeID, role, content string) error {
	interaction := models.MemeInteraction{
		UserID:    userID,
		MemeID:    memeID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	}

	return database.DB.Create(&interaction).Error
}

func (s *OpenAIService) GetMemeHistory(userID int64, memeID string) ([]models.MemeInteraction, error) {
	var interactions []models.MemeInteraction
	err := database.DB.Where("user_id = ? AND meme_id = ?", userID, memeID).
		Order("created_at ASC").
		Limit(20). // количество взаимодействий
		Find(&interactions).Error

	return interactions, err
}

func (s *OpenAIService) AddCommentMapping(messageID int, memeID string) error {
	mapping := models.CommentMemeMapping{
		MessageID: messageID,
		MemeID:    memeID,
		CreatedAt: time.Now(),
	}

	return database.DB.Create(&mapping).Error
}

func (s *OpenAIService) GetMemeIDByComment(messageID int) (string, error) {
	var mapping models.CommentMemeMapping
	err := database.DB.Where("message_id = ?", messageID).First(&mapping).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}

	return mapping.MemeID, nil
}

func (s *OpenAIService) GetRecentMemes(userID int64, limit int) ([]string, error) {
	type MemeResult struct {
		MemeID   string    `gorm:"column:meme_id"`
		LatestAt time.Time `gorm:"column:latest_at"`
	}

	var results []MemeResult

	err := database.DB.Model(&models.MemeInteraction{}).
		Select("meme_id, MAX(created_at) as latest_at").
		Where("user_id = ?", userID).
		Group("meme_id").
		Order("latest_at DESC").
		Limit(limit).
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	memeIDs := make([]string, len(results))
	for i, result := range results {
		memeIDs[i] = result.MemeID
	}

	return memeIDs, nil
}

func (s *OpenAIService) GetMemeContextualResponse(ctx context.Context, userID int64, memeID, query string) (string, error) {
	memeHistory, err := s.GetMemeHistory(userID, memeID)
	if err != nil {
		return "", err
	}

	if len(memeHistory) == 0 {
		return s.GetResponse(ctx, query, userID)
	}

	var contextualPrompt strings.Builder
	contextualPrompt.WriteString("Вот информация о меме и предыдущие комментарии к нему:\n\n")

	for _, entry := range memeHistory {
		if entry.Role == "user" && (strings.HasPrefix(entry.Content, "[MEME") || strings.HasPrefix(entry.Content, "[VIDEO")) {
			contextualPrompt.WriteString("Пользователь отправил мем\n")
		} else {
			roleName := "Пользователь"
			if entry.Role == "assistant" {
				roleName = "Я (Сталин)"
			}
			contextualPrompt.WriteString(fmt.Sprintf("%s: %s\n", roleName, entry.Content))
		}
	}

	contextualPrompt.WriteString(fmt.Sprintf("\nПользователь сейчас спрашивает: %s\n", query))
	contextualPrompt.WriteString("Ответь на комментарий пользователя, сохраняя свой характер Сталина и помня контекст мема.")

	response, err := s.GetResponse(ctx, contextualPrompt.String(), userID)
	if err != nil {
		return "", err
	}

	err = s.AddMemeInteraction(userID, memeID, "user", query)
	if err != nil {
		log.Printf("Error adding user meme interaction: %v", err)
	}

	err = s.AddMemeInteraction(userID, memeID, "assistant", response)
	if err != nil {
		log.Printf("Error adding assistant meme interaction: %v", err)
	}

	return response, nil
}

func (s *OpenAIService) GetMemesSummary(userID int64, memeIDs []string) map[string]string {
	summaries := make(map[string]string)

	for _, memeID := range memeIDs {
		history, err := s.GetMemeHistory(userID, memeID)
		if err != nil || len(history) == 0 {
			summaries[memeID] = "Мем не найден"
			continue
		}

		var botComment string
		for _, entry := range history {
			if entry.Role == "assistant" && len(entry.Content) > 0 {
				botComment = entry.Content
				break
			}
		}

		if botComment == "" {
			botComment = "Без комментария"
		}

		if len(botComment) > 100 {
			botComment = botComment[:100] + "..."
		}

		summaries[memeID] = botComment
	}

	return summaries
}

func (s *OpenAIService) ClearMemeHistory(userID int64) error {
	return database.DB.Where("user_id = ?", userID).Delete(&models.MemeInteraction{}).Error
}
