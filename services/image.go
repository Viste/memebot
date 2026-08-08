package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"memebot/metrics"

	"github.com/sashabaranov/go-openai"
)

// MemeIdea — придуманный ботом мем: промпт для генератора картинок и подпись-реплика.
type MemeIdea struct {
	ImagePrompt string `json:"image_prompt"`
	Caption     string `json:"caption"`
}

const memeIdeaInstruction = `Тебе нужно придумать СВОЙ мем. Ответь строго JSON-объектом без пояснений с двумя полями:
"image_prompt" — детальное описание картинки-мема на английском языке для генератора изображений: стиль, композиция, персонажи, атмосфера. Текст, который должен быть НА картинке, укажи на русском языке в кавычках и напиши, где он расположен. Мем должен быть смешным и в твоём духе.
"caption" — твоя короткая реплика к этому мему (1-2 предложения, сплошным текстом, в твоём обычном стиле).`

// GenerateMemeRemake придумывает и рисует собственный мем бота.
// sourceImageURLs — исходные мемы для контекста (может быть пусто),
// memeContext — текстовая история обсуждения мема,
// userRequest — что попросил пользователь (или внутренняя инструкция),
// trigger — метка для метрик: "request" или "random".
// Возвращает PNG-байты картинки и подпись.
func (s *OpenAIService) GenerateMemeRemake(ctx context.Context, sourceImageURLs []string, memeContext, userRequest, trigger string) ([]byte, string, error) {
	startTime := time.Now()

	idea, err := s.inventMemeIdea(ctx, sourceImageURLs, memeContext, userRequest)
	if err != nil && len(sourceImageURLs) > 0 {
		log.Printf("Meme idea with images failed, retrying text-only: %v", err)
		idea, err = s.inventMemeIdea(ctx, nil, memeContext, userRequest)
	}
	if err != nil {
		metrics.TrackImageError("idea_generation_failed")
		return nil, "", fmt.Errorf("meme idea error: %w", err)
	}

	imageData, err := s.renderMemeImage(ctx, idea.ImagePrompt)
	if err != nil {
		metrics.TrackImageError("image_render_failed")
		return nil, "", fmt.Errorf("image render error: %w", err)
	}

	duration := time.Since(startTime).Seconds()
	metrics.TrackImageGenerated(trigger, duration)
	metrics.TrackOpenAIRequest(s.imageModel, "meme_remake", duration)

	return imageData, idea.Caption, nil
}

// inventMemeIdea просит чат-модель придумать мем и вернуть JSON с промптом и подписью.
func (s *OpenAIService) inventMemeIdea(ctx context.Context, sourceImageURLs []string, memeContext, userRequest string) (*MemeIdea, error) {
	var userText strings.Builder
	if memeContext != "" {
		userText.WriteString("Контекст обсуждения мема:\n")
		userText.WriteString(memeContext)
		userText.WriteString("\n\n")
	}
	userText.WriteString(userRequest)

	parts := []openai.ChatMessagePart{
		{
			Type: openai.ChatMessagePartTypeText,
			Text: userText.String(),
		},
	}
	for _, url := range sourceImageURLs {
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
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: memeIdeaInstruction,
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
		ReasoningEffort:     s.getRandomReasoningEffort(),
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	if err != nil {
		metrics.TrackOpenAIError("api_request_failed")
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if resp.Usage.PromptTokens > 0 {
		metrics.TrackOpenAITokens(s.model, "prompt", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens > 0 {
		metrics.TrackOpenAITokens(s.model, "completion", resp.Usage.CompletionTokens)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from chat model")
	}

	var idea MemeIdea
	content := resp.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(content), &idea); err != nil {
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(content[start:end+1]), &idea); err2 != nil {
				return nil, fmt.Errorf("failed to parse meme idea JSON: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to parse meme idea JSON: %w", err)
		}
	}

	if idea.ImagePrompt == "" {
		return nil, fmt.Errorf("meme idea has empty image_prompt")
	}
	if idea.Caption == "" {
		idea.Caption = "Вот как надо делать мемы, товарищ."
	}

	return &idea, nil
}

// renderMemeImage генерирует картинку через Images API.
func (s *OpenAIService) renderMemeImage(ctx context.Context, prompt string) ([]byte, error) {
	resp, err := s.client.CreateImage(ctx, openai.ImageRequest{
		Model:   s.imageModel,
		Prompt:  prompt,
		N:       1,
		Size:    openai.CreateImageSize1024x1024,
		Quality: openai.CreateImageQualityMedium,
	})
	if err != nil {
		metrics.TrackOpenAIError("image_request_failed")
		return nil, fmt.Errorf("image API error: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("image API returned no data")
	}

	data := resp.Data[0]

	if data.B64JSON != "" {
		decoded, err := base64.StdEncoding.DecodeString(data.B64JSON)
		if err != nil {
			return nil, fmt.Errorf("failed to decode image base64: %w", err)
		}
		return decoded, nil
	}

	if data.URL != "" {
		return downloadImage(ctx, data.URL)
	}

	return nil, fmt.Errorf("image API returned neither b64 nor url")
}

func downloadImage(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image download failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(io.LimitReader(resp.Body, 20<<20))
}
