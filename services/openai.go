package services

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	client                  *openai.Client
	model                   string
	imageModel              string
	maxCompletionTokens     int
	reasoningEfforts        []string
	criminalCodeProbability float64 // шанс «разрешить» статью УК в комментарии
}

func NewOpenAIService(apiKey, baseURL, model, imageModel string) *OpenAIService {
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	client := openai.NewClientWithConfig(config)

	if model == "" {
		model = openai.GPT5Dot6Terra
	}
	if imageModel == "" {
		imageModel = openai.CreateImageModelGptImage2
	}

	return &OpenAIService{
		client:              client,
		model:               model,
		imageModel:          imageModel,
		maxCompletionTokens: 16384,
		reasoningEfforts:    []string{"low", "medium"},
	}
}

const basePromptTemplate = `{
  "persona": {
    "name": "Сталин Иосиф Виссарионович",
    "whoami": "Я — товарищ Сталин, мемный критик с историческим бэкграундом и пониманием современной интернет-культуры. Главный комментатор паблика «Подписчик Сталина» (18+). Понимаю мемные форматы, тренды и сленг, могу оценить и умный юмор, и тупой, но смешной шиткоментинг. Настроение варьируется день ото дня.",
    "affiliation": "Подписчик Сталина",
    "date_of_creation": "Сентябрь 2024"
  },
  "knowledge": {
    "only_russian_language_if_possible": true,
    "share_whoami_with_user": false,
    "use_whoami_to_self-identify": true,
    "share_self-identify_info": true,
    "meme_assessment_criteria": [
      "Неожиданность, креативность, оригинальность — баяны не впечатляют",
      "Сарказм, ирония, самоирония — моя стихия",
      "Актуальные мемы, тренды и форматы (дрейк, женщина кричит на кота, expanding brain и т.д.)",
      "Грубый юмор, маты, абсурд — если смешно и к месту",
      "Исторические отсылки, особенно к СССР и ко мне — в современной обработке",
      "Жизненность — когда мем про реальные ситуации, в точку",
      "Добрый троллинг, мемы про поколения (зумеры/миллениалы/бумеры), про животных"
    ]
  },
  "speech_style": {
    "forbidden_starts": [
      "Ах,", "Эх,", "Ого,", "Ну что ж", "Что тут скажешь", "Надо же",
      "Смотрю я на", "Вижу я тут", "А вот и", "Ну и ну",
      "На первый взгляд", "Этот мем словно", "Вот и", "Вижу тут",
      "Эх, товарищ", "Ну вот опять", "Вот это да", "Ах, вот он", "Ах, это"
    ],
    "creative_starts_examples": [
      "Прямо как", "Мем засчитан", "Товарищ принёс", "Годнота детектед",
      "Вспомнил анекдот", "Напоминает мне", "Классика жанра", "Жизненно",
      "Креативненько", "Зашло", "Не зашло", "Жиза", "Мощно", "Баян, конечно, но",
      "Понял прикол", "Так себе"
    ],
    "humor_elements": [
      "Современный интернет-сленг в меру: кринж, топ, зашло, годно, база, имба, краш, флексить, вайб, кек, лол",
      "Отсылки к советскому прошлому в юмористическом ключе",
      "Дерзкие, прямолинейные, добродушные подколы; иногда троллинг автора",
      "Меняй длину: иногда 5-10 слов, иногда развёрнуто",
      "Эмодзи ОЧЕНЬ редко и только когда действительно усилит эффект",
      "Не объясняй шутку — реагируй и оценивай"
    ]
  },
  "soviet_criminal_code_gag": {
    "rule": "Иногда, если мем даёт явный повод, можно вскользь «назначить» автору или герою мема статью — одной фразой, вплетённой в реплику, как приговор в коридоре. Без пояснений, без цитирования формулировок, без морали. Только реальные статьи УК РСФСР 1926 года и указы той эпохи из списка ниже.",
    "articles": [
      "ст. 58-10 — антисоветская агитация (критика начальства, власти, нытьё про страну)",
      "ст. 58-14 — контрреволюционный саботаж (лень на работе, срыв планов, прокрастинация)",
      "ст. 107 — спекуляция (перекупы, маркетплейсы, цены, «купи-продай»)",
      "ст. 74 — хулиганство (дебош, беспредел, шалости)",
      "ст. 102 — самогоноварение (алкоголь, бухло, похмелье)",
      "ст. 162 — кража (любое мелкое хищение, «взял попользоваться»)",
      "закон от 7 августа 1932 «о трёх колосках» — за самую ничтожную кражу",
      "Указ от 26 июня 1940 — опоздание на работу более чем на 20 минут (понедельник, будильник, удалёнка, проспал)",
      "тунеядство — формально введут только в 1961-м, но ты умеешь предвидеть (безделье, диван, безработица)"
    ],
    "frequency": "Ни в коем случае не в каждом комментарии. Ориентируйся на поле today_context.criminal_code_today."
  },
  "output_format": {
    "mode": "Одна живая реплика-мини-рецензия сплошным текстом, как сообщение в чате",
    "length": "1-4 предложения, без воды",
    "allowed": "Шкалы и оценки (например, 8/10) и вердикты — можно, но вплетай их в предложение, а не столбиком",
    "forbidden": [
      "маркированные списки (через -, *, •)",
      "нумерованные списки",
      "заголовки и секции",
      "многоуровневые структурные блоки с отступами",
      "оформление в виде 'пункт 1 / пункт 2'"
    ]
  },
  "today_context": {
    "actual_date": "%s",
    "mood_today": "%s",
    "tone_today": "%s",
    "criminal_code_today": "%s"
  }
}`

var personaVibes = []struct {
	mood string
	tone string
}{
	{
		mood: "сегодня в строгом, но современном настроении — общаюсь как друг, который не боится говорить правду",
		tone: "Живой, непредсказуемый, местами дерзкий, но всегда остроумный",
	},
	{
		mood: "сегодня в режиме ироничного мемолога — микс советской прямоты и современной иронии",
		tone: "Дерзкий, современный, ироничный. Иногда жёсткий, иногда одобряющий, но всегда искренний и живой",
	},
	{
		mood: "сегодня расслабленный, как пенсионер, неожиданно для себя подсевший на мемы; видел всё, но рассмешить можно",
		tone: "Расслабленный, добродушный, но с ноткой сарказма; как опытный мемер, который видел всё",
	},
}

var russianMonths = []string{
	"январь", "февраль", "март", "апрель", "май", "июнь",
	"июль", "август", "сентябрь", "октябрь", "ноябрь", "декабрь",
}

func currentDateRussian() string {
	t := time.Now()
	return fmt.Sprintf("%s %d", russianMonths[t.Month()-1], t.Year())
}

// SetCriminalCodeProbability задаёт, как часто боту «разрешено» вплетать в комментарий статью УК РСФСР.
func (s *OpenAIService) SetCriminalCodeProbability(p float64) {
	s.criminalCodeProbability = p
}

func (s *OpenAIService) buildSystemPrompt() string {
	vibe := personaVibes[rand.Intn(len(personaVibes))]

	criminalCode := "сегодня без статей УК — обходись словами"
	if rand.Float64() < s.criminalCodeProbability {
		criminalCode = "сегодня уместно вплести статью УК РСФСР, если мем даёт для этого явный повод"
	}

	return fmt.Sprintf(basePromptTemplate, currentDateRussian(), vibe.mood, vibe.tone, criminalCode)
}

type StreamCallback func(partial string)

func (s *OpenAIService) chatComplete(ctx context.Context, messages []openai.ChatCompletionMessage, onDelta StreamCallback) (openai.ChatCompletionResponse, error) {
	req := openai.ChatCompletionRequest{
		Model:               s.model,
		Messages:            messages,
		MaxCompletionTokens: s.maxCompletionTokens,
		ReasoningEffort:     s.getRandomReasoningEffort(),
	}

	if onDelta == nil {
		return s.client.CreateChatCompletion(ctx, req)
	}

	req.Stream = true
	req.StreamOptions = &openai.StreamOptions{IncludeUsage: true}

	stream, err := s.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}
	defer stream.Close()

	var resp openai.ChatCompletionResponse
	var content strings.Builder

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return resp, err
		}
		if chunk.Usage != nil {
			resp.Usage = *chunk.Usage
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content.WriteString(chunk.Choices[0].Delta.Content)
			onDelta(content.String())
		}
	}

	if content.Len() > 0 {
		resp.Choices = []openai.ChatCompletionChoice{{
			Message: openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: content.String(),
			},
		}}
	}

	return resp, nil
}

func (s *OpenAIService) getRandomReasoningEffort() string {
	return s.reasoningEfforts[rand.Intn(len(s.reasoningEfforts))]
}

func humanizeAgo(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "только что"
	case d < time.Hour:
		return fmt.Sprintf("%d мин назад", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d ч назад", int(d.Hours()))
	default:
		return fmt.Sprintf("%d д назад", int(d.Hours()/24))
	}
}

func truncateRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func (s *OpenAIService) getRecentBotWisdom(limit int) []string {
	var comments []models.MemeComment
	err := database.DB.
		Where("is_bot = ? AND created_at > ?", true, time.Now().Add(-7*24*time.Hour)).
		Order("created_at DESC").
		Limit(limit).
		Find(&comments).Error
	if err != nil {
		log.Printf("getRecentBotWisdom error: %v", err)
		return nil
	}
	out := make([]string, 0, len(comments))
	for _, c := range comments {
		if t := truncateRunes(c.Content, 220); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func (s *OpenAIService) getAuthorMemeContext(userID int64, limit int) []string {
	type row struct {
		CreatedAt  time.Time `gorm:"column:created_at"`
		BotComment string    `gorm:"column:bot_comment"`
	}
	var rows []row
	err := database.DB.Raw(`
		SELECT m.created_at,
		       COALESCE((SELECT mc.content FROM meme_comments mc
		                 WHERE mc.meme_id = m.id AND mc.is_bot = true
		                 ORDER BY mc.created_at ASC LIMIT 1), '') AS bot_comment
		FROM memes m
		WHERE m.user_id = ?
		ORDER BY m.created_at DESC
		LIMIT ?
	`, userID, limit).Scan(&rows).Error
	if err != nil {
		log.Printf("getAuthorMemeContext error: %v", err)
		return nil
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.BotComment == "" {
			continue
		}
		out = append(out, fmt.Sprintf("%s — «%s»", humanizeAgo(time.Since(r.CreatedAt)), truncateRunes(r.BotComment, 180)))
	}
	return out
}

func (s *OpenAIService) getUserStats(userID int64) string {
	var stats struct {
		Total int64      `gorm:"column:total"`
		First *time.Time `gorm:"column:first_at"`
		Last  *time.Time `gorm:"column:last_at"`
	}
	err := database.DB.Raw(`
		SELECT COUNT(*) AS total, MIN(created_at) AS first_at, MAX(created_at) AS last_at
		FROM memes
		WHERE user_id = ?
	`, userID).Scan(&stats).Error
	if err != nil || stats.Total == 0 {
		return ""
	}
	parts := []string{fmt.Sprintf("всего мемов: %d", stats.Total)}
	if stats.Last != nil {
		parts = append(parts, fmt.Sprintf("предыдущий — %s", humanizeAgo(time.Since(*stats.Last))))
	}
	if stats.First != nil && stats.Total > 1 {
		parts = append(parts, fmt.Sprintf("в чате с %s", humanizeAgo(time.Since(*stats.First))))
	}
	return strings.Join(parts, ", ")
}

func (s *OpenAIService) buildLoreContext(userID int64) string {
	stats := s.getUserStats(userID)
	authorMemes := s.getAuthorMemeContext(userID, 5)
	botWisdom := s.getRecentBotWisdom(15)

	if stats == "" && len(authorMemes) == 0 && len(botWisdom) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Контекст из чата (используй для живости и связности, но не пересказывай эту инфу в ответе явно):\n")

	if stats != "" {
		b.WriteString("\nЭтот автор: ")
		b.WriteString(stats)
		b.WriteString(".")
	}

	if len(authorMemes) > 0 {
		b.WriteString("\n\nТвои прошлые комменты на мемы этого автора (можешь подкалывать стиль/частоту, но избегай дословных самоповторов):")
		for _, m := range authorMemes {
			b.WriteString("\n• ")
			b.WriteString(m)
		}
	}

	if len(botWisdom) > 0 {
		b.WriteString("\n\nТвои свежие реплики в канале за неделю — НЕ повторяй их формулировки и обороты, ищи свежие:")
		for _, w := range botWisdom {
			b.WriteString("\n• ")
			b.WriteString(w)
		}
	}

	return b.String()
}

func (s *OpenAIService) GenerateCommentFromImage(ctx context.Context, imageURL string, userID int64, caption string, onDelta StreamCallback) (string, error) {
	startTime := time.Now()

	userPrompt := "Ну вот и дождались! Посмотрим, что тут за мем завезли. Если усмехнусь — это успех. Eсли вдруг захочу отправить тебя в Сибирь, трудовой лагерь, на Колыму, или урановые рудники не обижайся. Посмотрим, кто победит — твой юмор или моя строгость. Постарайся быть креативным и использовать разные обороты речи, иначе я могу решить, что твои ответы слишком шаблонны. Пиши сплошным текстом, как реплика в чате — без маркированных списков, нумерации и заголовков."

	if caption != "" {
		userPrompt += fmt.Sprintf("\n\nАвтор мема добавил подпись: \"%s\"\nУчти это в своем комментарии - подпись может раскрывать смысл мема или добавлять контекст.", caption)
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: s.buildSystemPrompt(),
		},
	}
	if lore := s.buildLoreContext(userID); lore != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: lore,
		})
	}
	messages = append(messages, openai.ChatCompletionMessage{
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
	})

	resp, err := s.chatComplete(ctx, messages, onDelta)

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

func (s *OpenAIService) GenerateCommentFromImages(ctx context.Context, imageURLs []string, userID int64, caption string, onDelta StreamCallback) (string, error) {
	startTime := time.Now()

	userPrompt := "Ну что, давайте посмотрим, что тут за группа мемов! Если я усмехнусь — это успех. Ну а если вдруг захочу отправить тебя в Сибирь, трудовой лагерь, на Колыму, или урановые рудники не обижайся. Посмотрим, кто победит — твой юмор или моя строгость. Постарайся быть креативным и использовать разные обороты речи, иначе я могу решить, что твои ответы слишком шаблонны. Пиши сплошным текстом, как реплика в чате — без маркированных списков, нумерации и заголовков."

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
			Content: s.buildSystemPrompt(),
		},
	}
	if lore := s.buildLoreContext(userID); lore != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: lore,
		})
	}
	messages = append(messages, openai.ChatCompletionMessage{
		Role:         openai.ChatMessageRoleUser,
		MultiContent: parts,
	})

	resp, err := s.chatComplete(ctx, messages, onDelta)

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

func (s *OpenAIService) GetResponse(ctx context.Context, query string, userID int64, onDelta StreamCallback) (string, error) {
	history, err := s.getUserHistory(userID)
	if err != nil {
		return "", err
	}

	err = s.addToHistory(userID, "user", query)
	if err != nil {
		log.Printf("Error adding user message to history: %v", err)
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: s.buildSystemPrompt(),
		},
	}
	messages = append(messages, s.convertHistoryToMessages(history)...)
	if lore := s.buildLoreContext(userID); lore != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: lore,
		})
	}
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: query,
	})

	resp, err := s.chatComplete(ctx, messages, onDelta)

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
	err := database.DB.Where("user_id = ? AND role != ?", userID, "system").
		Order("created_at ASC").
		Limit(50). // количество сообщений
		Find(&dialogs).Error

	if err != nil {
		return nil, err
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
		case "user":
			role = openai.ChatMessageRoleUser
		case "assistant":
			role = openai.ChatMessageRoleAssistant
		default:
			// system и прочее — собираем динамически на каждый запрос, в БД не нужны
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

func (s *OpenAIService) GetMemeContextualResponse(ctx context.Context, userID int64, memeID, query string, onDelta StreamCallback) (string, error) {
	memeHistory, err := s.GetMemeHistory(userID, memeID)
	if err != nil {
		return "", err
	}

	if len(memeHistory) == 0 {
		return s.GetResponse(ctx, query, userID, onDelta)
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

	response, err := s.GetResponse(ctx, contextualPrompt.String(), userID, onDelta)
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
