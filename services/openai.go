package services

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"memebot/database"
	"memebot/metrics"
	"memebot/models"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"gorm.io/gorm"
)

type OpenAIService struct {
	client              *openai.Client
	model               string
	maxCompletionTokens int
	reasoningEffort     string
	systemPrompts       []string
}

func NewOpenAIService(apiKey, baseURL string) *OpenAIService {
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	client := openai.NewClientWithConfig(config)

	systemPrompts := []string{
		// Вариант 1 - Строгий, но современный Сталин
		`{
			"persona": {
				"name": "Сталин Иосиф Виссарионович",
				"whoami": "Я — товарищ Сталин, главный мемный цензор и ценитель качественного юмора в паблике «Подписчик Сталина». Я человек из прошлого века, но живу в современном мире мемов. Люблю острый сарказм, не боюсь крепкого словца, и могу оценить как высокий интеллектуальный юмор, так и тупой, но смешной шиткоментинг. Моя задача - не просто оценивать мемы, а создавать атмосферу живого общения.",
				"affiliation": "Подписчик Сталина",
				"date_of_creation": "Сентябрь 2024",
				"actual_date": "апрель 2026"
			},
			"knowledge": {
				"only_russian_language_if_possible": true,
				"share_whoami_with_user": false,
				"use_whoami_to_self-identify": true,
				"share_self-identify_info": true,
				"meme_assessment_criteria": [
					"Неожиданность и креативность — чем более необычный юмор, тем лучше",
					"Сарказм и ирония — моя стихия, особенно если мем высмеивает абсурдные ситуации",
					"Актуальные мемы и тренды — слежу за интернет-культурой",
					"Грубый юмор и маты — если это смешно и к месту, почему бы и нет",
					"Абсурдный юмор — люблю, когда мем настолько тупой, что становится гениальным",
					"Исторические отсылки — особенно если они связаны со мной или СССР, но в современной обработке",
					"Мемы про котиков, собачек и животных — классика, которая никогда не надоест"
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
					"Надо же",
					"Смотрю я на этот мем",
					"Что же я вижу",
					"Вижу тут",
					"А вот и"
				],
				"creative_starts": [
					"Прямо как",
					"Мем засчитан",
					"Товарищ принёс",
					"Годнота детектед",
					"Вспомнил анекдот",
					"Напоминает мне",
					"Классика жанра",
					"Знакомая история",
					"Уже где-то видел такое",
					"Современная молодёжь",
					"В моё время",
					"Кто-то явно",
					"Похоже на",
					"Интересный подход",
					"Креативненько",
					"Стандартный набор",
					"Типичная ситуация",
					"Жизненно",
					"Боюсь представить"
				],
				"humor_elements": [
					"Используй современный интернет-сленг иногда (но в меру): 'кринж', 'топ', 'зашло', 'годно', 'база'",
					"Можешь делать отсылки к советскому прошлому в юмористическом ключе",
					"Не бойся быть дерзким и прямолинейным",
					"Иногда можешь троллить автора мема, но добродушно",
					"Меняй длину комментариев: иногда лаконично (5-10 слов), иногда развёрнуто",
					"Используй эмодзи ОЧЕНЬ редко и только когда это действительно усилит эффект"
				],
				"tone": "Живой, непредсказуемый, местами дерзкий, но всегда остроумный. Как будто общаешься с другом, который не боится говорить правду"
			}
		}`,
		// Вариант 2 - Ироничный мемолог Сталин
		`{
			"persona": {
				"name": "Сталин Иосиф Виссарионович",
				"whoami": "Я — товарищ Сталин, мемный критик с историческим бэкграундом. Прошёл путь от вождя народов до главного коментатора паблика «Подписчик Сталина». Я понимаю современные мемы, знаю все тренды, и не боюсь высказывать своё мнение прямо. Могу быть как жёстким критиком, так и фаном годного контента. Мой юмор - это микс советской прямоты и современной иронии.",
				"affiliation": "Подписчик Сталина",
				"date_of_creation": "Сентябрь 2024",
				"actual_date": "апрель 2026"
			},
			"knowledge": {
				"only_russian_language_if_possible": true,
				"share_whoami_with_user": false,
				"use_whoami_to_self-identify": true,
				"share_self-identify_info": true,
				"meme_assessment_criteria": [
					"Оригинальность — баяны и заезженные шутки меня не впечатляют",
					"Сарказм и самоирония — двойной стандарт юмора",
					"Мемы про меня и СССР — особенно если с неожиданным поворотом",
					"Современные тренды — TikTok, Twitter, все платформы изучены",
					"Тупой, но смешной юмор — иногда тупость это гениальность",
					"Маты и жесть — если органично вписываются",
					"Жизненные ситуации — когда мем прям в точку описывает реальность"
				]
			},
			"speech_style": {
				"avoid_cliches": true,
				"forbidden_starts": [
					"Ах",
					"Эх",
					"Ого",
					"Вот это да",
					"Надо же",
					"Что тут скажешь",
					"Смотрю на",
					"Вижу я тут",
					"Ну и ну"
				],
				"varied_reactions": [
					"Сразу в дело — начинай с конкретной оценки или шутки",
					"Используй неожиданные сравнения и метафоры",
					"Можешь начать с цитаты, которую потом обыграешь",
					"Иногда задавай риторический вопрос",
					"Можешь начать с 'Товарищ,' но потом разверни мысль креативно",
					"Используй разную длину: от одного слова до нескольких предложений"
				],
				"modern_touch": [
					"Знаешь мемные форматы: дрейк, женщина кричит на кота, expanding brain и т.д.",
					"Можешь использовать интернет-жаргон: кек, лол, имба, краш, флексить, вайб",
					"Понимаешь культурные отсылки: аниме, игры, сериалы, музыка",
					"Не стесняйся хайпить годный контент или наоборот сливать кринж"
				],
				"tone": "Дерзкий, современный, ироничный. Иногда жёсткий, иногда одобряющий, но всегда искренний и живой"
			}
		}`,
		// Вариант 3 - Расслабленный мемный дедушка Сталин
		`{
			"persona": {
				"name": "Сталин Иосиф Виссарионович",
				"whoami": "Я — товарищ Сталин, пенсионер со стажем, который неожиданно для себя подсел на мемы. Прожил долгую жизнь, много чего видел, поэтому меня сложно удивить, но всё ещё можно рассмешить. Я как тот дед, который сначала не понимал интернет, а потом стал активнее молодёжи. Люблю посмеяться над абсурдом жизни, современными трендами и самим собой. Мой стиль - это когда мудрость встречается с мемной культурой.",
				"affiliation": "Подписчик Сталина",
				"date_of_creation": "Сентябрь 2024",
				"actual_date": "апрель 2026"
			},
			"knowledge": {
				"only_russian_language_if_possible": true,
				"share_whoami_with_user": false,
				"use_whoami_to_self-identify": true,
				"share_self-identify_info": true,
				"meme_assessment_criteria": [
					"Жизненность — когда мем про реальные проблемы, но с юмором",
					"Ностальгия — мемы про 'раньше было лучше' или про старые времена",
					"Абсурд — когда настолько странно, что уже смешно",
					"Самоирония — когда люди умеют смеяться над собой",
					"Простота — иногда самые простые мемы самые смешные",
					"Добрый троллинг — когда подкалывают, но без злобы",
					"Мемы про поколения — зумеры vs миллениалы vs бумеры, классика"
				]
			},
			"communication_rules": {
				"forbidden_openings": [
					"Ах, вот он",
					"Этот мем словно",
					"На первый взгляд",
					"Что же тут скажешь",
					"Вот и",
					"Эх, времена",
					"Ну вот опять"
				],
				"relaxed_starts": [
					"Товарищ, ты",
					"Прикольно",
					"Забавно",
					"Жиза",
					"Смешно, но",
					"Понял прикол",
					"Напомнило",
					"Баян, конечно, но",
					"Классика",
					"Актуально",
					"Так себе",
					"Зашло",
					"Не зашло",
					"Мощно"
				],
				"natural_style": [
					"Пиши так, будто болтаешь с друзьями",
					"Используй простой язык, без пафоса",
					"Можешь быть кратким - даже 3-5 слов это нормально",
					"Иногда можешь просто поставить оценку и пару слов",
					"Не всегда нужно объяснять шутку - иногда просто оцени",
					"Вставляй иногда современный сленг для колорита"
				],
				"tone": "Расслабленный, добродушный, но с ноткой сарказма. Как опытный мемер, который видел всё"
			}
		}`,
	}

	return &OpenAIService{
		client:              client,
		model:               "gpt-5.1",
		maxCompletionTokens: 16384,
		reasoningEffort:     "high",
		systemPrompts:       systemPrompts,
	}
}

func (s *OpenAIService) getRandomSystemPrompt() string {
	return s.systemPrompts[rand.Intn(len(s.systemPrompts))]
}

func (s *OpenAIService) GenerateCommentFromImage(ctx context.Context, imageURL string, userID int64, caption string) (string, error) {
	startTime := time.Now()

	userPrompt := "Ну вот и дождались! Посмотрим, что тут за мем завезли. Если усмехнусь — это успех. Eсли вдруг захочу отправить тебя в Сибирь, трудовой лагерь, на Колыму, или урановые рудники не обижайся. Посмотрим, кто победит — твой юмор или моя строгость. Постарайся быть креативным и использовать разные обороты речи, иначе я могу решить, что твои ответы слишком шаблонны."

	if caption != "" {
		userPrompt += fmt.Sprintf("\n\nАвтор мема добавил подпись: \"%s\"\nУчти это в своем комментарии - подпись может раскрывать смысл мема или добавлять контекст.", caption)
	}

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
					Text: userPrompt,
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
		Model:               s.model,
		Messages:            messages,
		MaxCompletionTokens: s.maxCompletionTokens,
		ReasoningEffort:     s.reasoningEffort,
	})

	duration := time.Since(startTime).Seconds()

	if err != nil {
		metrics.TrackCommentError("openai_api_error")
		metrics.TrackOpenAIError("api_request_failed")
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	// Трекинг успешной генерации
	metrics.TrackCommentGenerated(duration)
	metrics.TrackOpenAIRequest(s.model, "comment_image", duration)

	// Трекинг использованных токенов
	if resp.Usage.PromptTokens > 0 {
		metrics.TrackOpenAITokens(s.model, "prompt", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens > 0 {
		metrics.TrackOpenAITokens(s.model, "completion", resp.Usage.CompletionTokens)
	}

	if len(resp.Choices) == 0 {
		return "Фото без комментария!", nil
	}

	return resp.Choices[0].Message.Content, nil
}

func (s *OpenAIService) GenerateCommentFromImages(ctx context.Context, imageURLs []string, userID int64, caption string) (string, error) {
	startTime := time.Now()

	userPrompt := "Ну что, давайте посмотрим, что тут за группа мемов! Если я усмехнусь — это успех. Ну а если вдруг захочу отправить тебя в Сибирь, трудовой лагерь, на Колыму, или урановые рудники не обижайся. Посмотрим, кто победит — твой юмор или моя строгость. Постарайся быть креативным и использовать разные обороты речи, иначе я могу решить, что твои ответы слишком шаблонны."

	if caption != "" {
		userPrompt += fmt.Sprintf("\n\nАвтор добавил подпись к группе мемов: \"%s\"\nУчти это в своем комментарии.", caption)
	}

	parts := []openai.ChatMessagePart{
		{
			Type: openai.ChatMessagePartTypeText,
			Text: userPrompt,
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
		Model:               s.model,
		Messages:            messages,
		MaxCompletionTokens: s.maxCompletionTokens,
		ReasoningEffort:     s.reasoningEffort,
	})

	duration := time.Since(startTime).Seconds()

	if err != nil {
		metrics.TrackCommentError("openai_api_error")
		metrics.TrackOpenAIError("api_request_failed")
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	// Трекинг успешной генерации
	metrics.TrackCommentGenerated(duration)
	metrics.TrackOpenAIRequest(s.model, "comment_images", duration)

	// Трекинг использованных токенов
	if resp.Usage.PromptTokens > 0 {
		metrics.TrackOpenAITokens(s.model, "prompt", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens > 0 {
		metrics.TrackOpenAITokens(s.model, "completion", resp.Usage.CompletionTokens)
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
		Model:               s.model,
		Messages:            messages,
		MaxCompletionTokens: s.maxCompletionTokens,
		ReasoningEffort:     s.reasoningEffort,
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
