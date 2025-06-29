package handlers

import (
	"context"
	"fmt"
	"log"
	"memebot/config"
	"memebot/services"
	"memebot/utils"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotHandlers struct {
	bot           *tgbotapi.BotAPI
	openaiService *services.OpenAIService
	mediaManager  *utils.MediaGroupManager
	config        *config.Config
}

func NewBotHandlers(bot *tgbotapi.BotAPI, openaiService *services.OpenAIService, cfg *config.Config) *BotHandlers {
	return &BotHandlers{
		bot:           bot,
		openaiService: openaiService,
		mediaManager:  utils.NewMediaGroupManager(),
		config:        cfg,
	}
}

func (h *BotHandlers) HandleUpdate(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	message := update.Message

	log.Printf("Received message from %s (%d) in chat %s (%d): %s",
		message.From.UserName, message.From.ID,
		message.Chat.Title, message.Chat.ID,
		message.Text)

	if message.IsCommand() {
		h.handleCommand(message)
		return
	}

	if message.Chat.IsPrivate() {
		h.handlePrivateMessage(message)
	} else if message.Chat.IsGroup() || message.Chat.IsSuperGroup() {
		h.handleGroupMessage(message)
	}
}

func (h *BotHandlers) handleCommand(message *tgbotapi.Message) {
	command := message.Command()

	switch command {
	case "start":
		h.handleStartCommand(message)
	case "memes", "meme":
		h.handleMemesCommand(message)
	case "forget":
		h.handleForgetCommand(message)
	case "help":
		h.handleHelpCommand(message)
	default:
		log.Printf("Unknown command: %s", command)
	}
}

func (h *BotHandlers) handleStartCommand(message *tgbotapi.Message) {
	if !message.Chat.IsPrivate() {
		return
	}

	firstName := message.From.FirstName
	if firstName == "" {
		firstName = "друг"
	}

	text := fmt.Sprintf("Привет %s, тут ты можешь отправить нам мемес. Принимаю только видосики и картинощки", firstName)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyToMessageID = message.MessageID

	_, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Error sending start message: %v", err)
	}
}

func (h *BotHandlers) handleMemesCommand(message *tgbotapi.Message) {
	if message.Chat.IsPrivate() {
		return
	}

	if !utils.IsValidChat(message.Chat.Title) {
		h.sendInvalidChatMessage(message)
		return
	}

	recentMemes, err := h.openaiService.GetRecentMemes(message.Chat.ID, 5)
	if err != nil {
		log.Printf("Error getting recent memes: %v", err)
		utils.SendReply(h.bot, message, "Произошла ошибка при получении списка мемов.")
		return
	}

	if len(recentMemes) == 0 {
		utils.SendReply(h.bot, message, "В этом чате ещё нет мемов, которые я помню.")
		return
	}

	summaries := h.openaiService.GetMemesSummary(message.Chat.ID, recentMemes)

	var response strings.Builder
	response.WriteString("Последние мемы в этом чате:\n\n")

	for i, memeID := range recentMemes {
		summary := summaries[memeID]
		response.WriteString(fmt.Sprintf("%d. %s\n", i+1, summary))
	}

	response.WriteString("\nОтвечайте на мои комментарии к мемам, и я буду помнить их контекст!")

	utils.SendReply(h.bot, message, response.String())
}

func (h *BotHandlers) handleForgetCommand(message *tgbotapi.Message) {
	if message.Chat.IsPrivate() {
		return
	}

	if !utils.IsValidChat(message.Chat.Title) {
		h.sendInvalidChatMessage(message)
		return
	}

	chatMember, err := h.bot.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: message.Chat.ID,
			UserID: message.From.ID,
		},
	})

	if err != nil {
		log.Printf("Error getting chat member: %v", err)
		utils.SendReply(h.bot, message, "Не удалось проверить права доступа.")
		return
	}

	if chatMember.Status != "administrator" && chatMember.Status != "creator" {
		utils.SendReply(h.bot, message, "Только администраторы могут использовать эту команду.")
		return
	}

	err = h.openaiService.ClearMemeHistory(message.Chat.ID)
	if err != nil {
		log.Printf("Error clearing meme history: %v", err)
		utils.SendReply(h.bot, message, "Произошла ошибка при очистке истории.")
		return
	}

	utils.SendReply(h.bot, message, "История мемов в этом чате очищена.")
}

func (h *BotHandlers) handleHelpCommand(message *tgbotapi.Message) {
	helpText := `🤖 Команды бота:

**Для личных сообщений:**
/start - Приветствие и инструкции
📷 Отправь фото - Переслать мем в канал
🎥 Отправь видео - Переслать мем в канал

**Для групп:**
/memes - Показать последние мемы в чате
/forget - Очистить историю мемов (только админы)
📷 Отправь фото - Получить комментарий от Сталина
💬 Ответь на комментарий бота - Продолжить диалог

Отправляй мемы и получай саркастичные комментарии от товарища Сталина! 😄`

	utils.SendReply(h.bot, message, helpText)
}

func (h *BotHandlers) handlePrivateMessage(message *tgbotapi.Message) {
	if h.config.IsUserBanned(message.From.ID) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "не хочу с тобой разговаривать")
		msg.ReplyToMessageID = message.MessageID
		h.bot.Send(msg)
		return
	}

	if len(message.Photo) > 0 {
		h.handlePrivatePhoto(message)
		return
	}

	if message.Video != nil {
		h.handlePrivateVideo(message)
		return
	}
}

func (h *BotHandlers) handlePrivatePhoto(message *tgbotapi.Message) {
	firstName, lastName := utils.GetSenderName(message)
	caption := fmt.Sprintf("Мем прислал: %s %s", firstName, lastName)

	if len(message.Photo) == 0 {
		log.Printf("No photos in message")
		return
	}

	photo := message.Photo[len(message.Photo)-1]

	if message.MediaGroupID != "" {
		h.handleMediaGroup(message.MediaGroupID, photo.FileID, caption, message)
	} else {
		h.sendSinglePhoto(photo.FileID, caption, message)
	}
}

func (h *BotHandlers) handlePrivateVideo(message *tgbotapi.Message) {
	firstName, lastName := utils.GetSenderName(message)
	caption := fmt.Sprintf("Мем прислал: %s %s", firstName, lastName)

	err := utils.SendVideoToChannel(h.bot, h.config.Channel, tgbotapi.FileID(message.Video.FileID), caption)
	if err != nil {
		log.Printf("Error sending video to channel: %v", err)
		utils.SendReply(h.bot, message, "Произошла ошибка при отправке видео.")
		return
	}

	utils.SendReply(h.bot, message, "Спасибо за мем! Пока-пока")
}

func (h *BotHandlers) handleMediaGroup(groupID, fileID, caption string, message *tgbotapi.Message) {
	media := tgbotapi.NewInputMediaPhoto(tgbotapi.FileID(fileID))

	h.mediaManager.AddToGroup(groupID, media, caption, message.Chat.ID)

	h.mediaManager.SetTimer(groupID, 5*time.Second, func(gID string) {
		h.processMediaGroup(gID, message)
	})
}

func (h *BotHandlers) processMediaGroup(groupID string, originalMessage *tgbotapi.Message) {
	group, exists := h.mediaManager.GetGroup(groupID)
	if !exists {
		log.Printf("Media group %s not found", groupID)
		return
	}

	defer h.mediaManager.RemoveGroup(groupID)

	err := utils.SendMediaGroupToChannel(h.bot, h.config.Channel, group.Media)
	if err != nil {
		log.Printf("Error sending media group to channel: %v", err)
		utils.SendReply(h.bot, originalMessage, "Произошла ошибка при отправке медиа группы.")
		return
	}

	utils.SendReply(h.bot, originalMessage, "Спасибо за мем! Приходи еще")
}

func (h *BotHandlers) sendSinglePhoto(fileID, caption string, message *tgbotapi.Message) {
	err := utils.SendToChannel(h.bot, h.config.Channel, tgbotapi.FileID(fileID), caption)
	if err != nil {
		log.Printf("Error sending photo to channel: %v", err)
		utils.SendReply(h.bot, message, "Произошла ошибка при отправке фото.")
		return
	}

	utils.SendReply(h.bot, message, "Спасибо за мем! Пока-пока")
}

func (h *BotHandlers) handleGroupMessage(message *tgbotapi.Message) {
	if !utils.IsValidChat(message.Chat.Title) {
		h.sendInvalidChatMessage(message)
		return
	}

	// Обрабатываем фото в группе
	if len(message.Photo) > 0 {
		h.handleGroupPhoto(message)
		return
	}

	if message.ReplyToMessage != nil && message.ReplyToMessage.From.IsBot {
		h.handleReplyToBot(message)
		return
	}

	log.Printf("Group message from %s %s: %s",
		message.From.FirstName, message.From.UserName, message.Text)
}

func (h *BotHandlers) handleGroupPhoto(message *tgbotapi.Message) {
	ctx := context.Background()

	if message.MediaGroupID != "" {
		h.handleCommentMediaGroup(message.MediaGroupID, message)
	} else {
		h.handleSinglePhotoComment(ctx, message)
	}
}

func (h *BotHandlers) handleCommentMediaGroup(groupID string, message *tgbotapi.Message) {
	h.mediaManager.AddToGroup(groupID, message, "", message.Chat.ID)

	h.mediaManager.SetTimer(groupID, 5*time.Second, func(gID string) {
		h.processCommentMediaGroup(gID)
	})
}

func (h *BotHandlers) processCommentMediaGroup(groupID string) {
	group, exists := h.mediaManager.GetGroup(groupID)
	if !exists {
		log.Printf("Comment media group %s not found", groupID)
		return
	}

	defer h.mediaManager.RemoveGroup(groupID)

	messages := make([]*tgbotapi.Message, 0, len(group.Media))
	for _, item := range group.Media {
		if msg, ok := item.(*tgbotapi.Message); ok {
			messages = append(messages, msg)
		}
	}

	if len(messages) == 0 {
		log.Printf("No messages found in group %s", groupID)
		return
	}

	ctx := context.Background()

	imageURLs := make([]string, 0, len(messages))
	for _, msg := range messages {
		if len(msg.Photo) > 0 {
			photo := msg.Photo[len(msg.Photo)-1]
			file, err := h.bot.GetFile(tgbotapi.FileConfig{FileID: photo.FileID})
			if err != nil {
				log.Printf("Error getting file info: %v", err)
				continue
			}

			imageURL := utils.GetImageURL(h.bot.Token, file.FilePath)
			imageURLs = append(imageURLs, imageURL)
		}
	}

	if len(imageURLs) == 0 {
		log.Printf("No valid image URLs found for group %s", groupID)
		return
	}

	comment, err := h.openaiService.GenerateCommentFromImages(ctx, imageURLs, messages[0].Chat.ID)
	if err != nil {
		log.Printf("Error generating comment for images: %v", err)
		return
	}

	sentMessage, err := utils.SendReply(h.bot, messages[0], comment)
	if err != nil {
		log.Printf("Error sending comment: %v", err)
		return
	}

	groupMemeID := fmt.Sprintf("group_%s", groupID)
	imageContent := fmt.Sprintf("[MEME_GROUP: %s]", strings.Join(imageURLs, ", "))

	h.openaiService.AddMemeInteraction(messages[0].Chat.ID, groupMemeID, "user", imageContent)
	h.openaiService.AddMemeInteraction(messages[0].Chat.ID, groupMemeID, "assistant", comment)

	if sentMessage != nil {
		h.openaiService.AddCommentMapping(sentMessage.MessageID, groupMemeID)
	}
}

func (h *BotHandlers) handleSinglePhotoComment(ctx context.Context, message *tgbotapi.Message) {
	if len(message.Photo) == 0 {
		log.Printf("No photos in message")
		return
	}

	photo := message.Photo[len(message.Photo)-1]
	file, err := h.bot.GetFile(tgbotapi.FileConfig{FileID: photo.FileID})
	if err != nil {
		log.Printf("Error getting file info: %v", err)
		utils.SendReply(h.bot, message, "Не удалось получить информацию о фотографии. Попробуйте еще раз.")
		return
	}

	imageURL := utils.GetImageURL(h.bot.Token, file.FilePath)
	log.Printf("Image URL: %s", imageURL)

	comment, err := h.openaiService.GenerateCommentFromImage(ctx, imageURL, message.Chat.ID)
	if err != nil {
		log.Printf("Error generating comment for single photo: %v", err)
		utils.SendReply(h.bot, message, "Не удалось обработать фотографию. Попробуйте еще раз.")
		return
	}

	sentMessage, err := utils.SendReply(h.bot, message, comment)
	if err != nil {
		log.Printf("Error sending comment: %v", err)
		return
	}

	memeID := photo.FileID
	imageContent := fmt.Sprintf("[MEME_IMAGE: %s]", imageURL)

	h.openaiService.AddMemeInteraction(message.Chat.ID, memeID, "user", imageContent)
	h.openaiService.AddMemeInteraction(message.Chat.ID, memeID, "assistant", comment)

	if sentMessage != nil {
		h.openaiService.AddCommentMapping(sentMessage.MessageID, memeID)
	}

	log.Printf("Generated comment: %s", comment)
}

func (h *BotHandlers) handleReplyToBot(message *tgbotapi.Message) {
	ctx := context.Background()

	memeID, err := h.openaiService.GetMemeIDByComment(message.ReplyToMessage.MessageID)
	if err != nil {
		log.Printf("Error getting meme ID by comment: %v", err)
		utils.SendReply(h.bot, message, "Не удалось обработать ваш запрос. Попробуйте позже.")
		return
	}

	var response string

	if memeID != "" {
		response, err = h.openaiService.GetMemeContextualResponse(ctx, message.Chat.ID, memeID, message.Text)
	} else {
		response, err = h.openaiService.GetResponse(ctx, message.Text, message.Chat.ID)
	}

	if err != nil {
		log.Printf("Error getting AI response: %v", err)
		utils.SendReply(h.bot, message, "Не удалось обработать ваш запрос. Попробуйте позже.")
		return
	}

	sentMessage, err := utils.SendReply(h.bot, message, response)
	if err != nil {
		log.Printf("Error sending AI response: %v", err)
		return
	}

	if memeID != "" && sentMessage != nil {
		h.openaiService.AddCommentMapping(sentMessage.MessageID, memeID)
	}
}

func (h *BotHandlers) sendInvalidChatMessage(message *tgbotapi.Message) {
	text := "Хорошая попытка, но я сделан только для паблика @stalinfollower"
	utils.SendReply(h.bot, message, text)
}
