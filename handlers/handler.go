package handlers

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"memebot/config"
	"memebot/metrics"
	"memebot/models"
	"memebot/services"
	"memebot/utils"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

type BotHandlers struct {
	bot           *tgbotapi.BotAPI
	openaiService *services.OpenAIService
	banService    *services.BanService
	mediaManager  *utils.MediaGroupManager
	config        *config.Config
}

func NewBotHandlers(bot *tgbotapi.BotAPI, openaiService *services.OpenAIService, banService *services.BanService, cfg *config.Config) *BotHandlers {
	return &BotHandlers{
		bot:           bot,
		openaiService: openaiService,
		banService:    banService,
		mediaManager:  utils.NewMediaGroupManager(),
		config:        cfg,
	}
}

func (h *BotHandlers) HandleUpdate(update tgbotapi.Update) {
	if update.ChannelPost != nil {
		h.handleChannelPost(update.ChannelPost)
		return
	}

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

// handleChannelPost запоминает мемы, которые админы постят в канал напрямую
func (h *BotHandlers) handleChannelPost(post *tgbotapi.Message) {
	if !h.isOurChannel(post.Chat) {
		return
	}

	var from *tgbotapi.User
	if post.AuthorSignature != "" {
		from = &tgbotapi.User{FirstName: post.AuthorSignature}
	}

	hasCaption := post.Caption != ""

	switch {
	case len(post.Photo) > 0:
		photo := post.Photo[len(post.Photo)-1]
		services.RecordChannelMeme(photo.FileID, "photo", from, post.MediaGroupID)
		metrics.TrackMemeReceived("channel_photo", hasCaption)
		log.Printf("Recorded channel photo post %d by %q", post.MessageID, post.AuthorSignature)
	case post.Video != nil:
		services.RecordChannelMeme(post.Video.FileID, "video", from, post.MediaGroupID)
		metrics.TrackMemeReceived("channel_video", hasCaption)
		log.Printf("Recorded channel video post %d by %q", post.MessageID, post.AuthorSignature)
	}
}

// isOurChannel сверяет чат с настроенным каналом (числовой ID или @username).
func (h *BotHandlers) isOurChannel(chat tgbotapi.Chat) bool {
	if chatID, err := strconv.ParseInt(h.config.Channel, 10, 64); err == nil {
		return chat.ID == chatID
	}
	return chat.UserName != "" && "@"+chat.UserName == h.config.Channel
}

func (h *BotHandlers) handleCommand(message *tgbotapi.Message) {
	command := message.Command()

	// Трекинг выполнения команды
	metrics.TrackCommandExecuted(command)

	switch command {
	case "start":
		h.handleStartCommand(message)
	case "memes", "meme":
		h.handleMemesCommand(message)
	case "forget":
		h.handleForgetCommand(message)
	case "help":
		h.handleHelpCommand(message)
	case "migrate_bans":
		h.handleMigrateBansCommand(message)
	case "ban":
		h.handleBanCommand(message)
	case "unban":
		h.handleUnbanCommand(message)
	case "banlist":
		h.handleBanlistCommand(message)
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
	msg.ReplyParameters = tgbotapi.ReplyParameters{MessageID: message.MessageID}

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
			ChatConfig: tgbotapi.ChatConfig{ChatID: message.Chat.ID},
			UserID:     message.From.ID,
		},
	})

	if err != nil {
		log.Printf("Error getting chat member: %v", err)
		utils.SendEphemeralReply(h.bot, message, "Не удалось проверить права доступа.")
		return
	}

	if chatMember.Status != "administrator" && chatMember.Status != "creator" {
		utils.SendEphemeralReply(h.bot, message, "Только администраторы могут использовать эту команду.")
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

**В личных сообщениях:**
/start - Приветствие и инструкции
📷 Принимаю картиночки(мемы)  - запощу мем в канале
🎥 Принимаю видео(мемы) - запощу мем в канале

**В группе:**
/memes - Показать последние мемы в чате
/forget - Очистить историю мемов (только админы)
📷 Отправь фото - Получить комментарий от Сталина
💬 Ответь на комментарий бота - Продолжить диалог
🎨 Ответь боту "а как бы ты сделал этот мем" - Сталин нарисует свою версию

**Управление банами (только админы):**
/migrate_bans - Перенести баны из конфига в БД
/ban <user_id> <причина> - Забанить пользователя
/unban <user_id> - Разбанить пользователя  
/banlist - Показать список забаненных

Отправляй мемы и получай саркастичные комментарии от товарища Сталина! 😄`

	utils.SendEphemeralReply(h.bot, message, helpText)
}

func (h *BotHandlers) handleMigrateBansCommand(message *tgbotapi.Message) {
	if !h.config.IsUserAdmin(message.From.ID) {
		utils.SendEphemeralReply(h.bot, message, "Только администраторы могут использовать эту команду.")
		return
	}

	err := h.banService.MigrateConfigBans(h.config.BannedUserIDs)
	if err != nil {
		log.Printf("Error migrating bans: %v", err)
		utils.SendReply(h.bot, message, "Произошла ошибка при миграции банов.")
		return
	}

	utils.SendReply(h.bot, message, fmt.Sprintf("Успешно мигрировано %d банов из конфига в базу данных.", len(h.config.BannedUserIDs)))
}

func (h *BotHandlers) handleBanCommand(message *tgbotapi.Message) {
	if !h.config.IsUserAdmin(message.From.ID) {
		utils.SendEphemeralReply(h.bot, message, "Только администраторы могут использовать эту команду.")
		return
	}

	args := strings.Fields(message.Text)
	if len(args) < 3 {
		utils.SendEphemeralReply(h.bot, message, "Использование: /ban <user_id> <причина>")
		return
	}

	userID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		utils.SendEphemeralReply(h.bot, message, "Неверный ID пользователя.")
		return
	}

	reason := strings.Join(args[2:], " ")

	err = h.banService.BanUser(userID, message.From.ID, reason)
	if err != nil {
		log.Printf("Error banning user %d: %v", userID, err)
		utils.SendReply(h.bot, message, fmt.Sprintf("Ошибка при бане пользователя: %v", err))
		return
	}

	metrics.TrackUserBanned()
	utils.SendReply(h.bot, message, fmt.Sprintf("Пользователь %d забанен. Причина: %s", userID, reason))
}

func (h *BotHandlers) handleUnbanCommand(message *tgbotapi.Message) {
	if !h.config.IsUserAdmin(message.From.ID) {
		utils.SendEphemeralReply(h.bot, message, "Только администраторы могут использовать эту команду.")
		return
	}

	args := strings.Fields(message.Text)
	if len(args) < 2 {
		utils.SendEphemeralReply(h.bot, message, "Использование: /unban <user_id>")
		return
	}

	userID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		utils.SendEphemeralReply(h.bot, message, "Неверный ID пользователя.")
		return
	}

	err = h.banService.UnbanUser(userID)
	if err != nil {
		log.Printf("Error unbanning user %d: %v", userID, err)
		utils.SendReply(h.bot, message, fmt.Sprintf("Ошибка при разбане: %v", err))
		return
	}

	metrics.TrackUserUnbanned()
	utils.SendReply(h.bot, message, fmt.Sprintf("Пользователь %d разбанен.", userID))
}

func (h *BotHandlers) handleBanlistCommand(message *tgbotapi.Message) {
	if !h.config.IsUserAdmin(message.From.ID) {
		utils.SendEphemeralReply(h.bot, message, "Только администраторы могут использовать эту команду.")
		return
	}

	bans, err := h.banService.GetBannedUsers(20)
	if err != nil {
		log.Printf("Error getting banned users: %v", err)
		utils.SendEphemeralReply(h.bot, message, "Ошибка при получении списка банов.")
		return
	}

	if len(bans) == 0 {
		utils.SendEphemeralReply(h.bot, message, "Список банов пуст.")
		return
	}

	var response strings.Builder
	response.WriteString("📛 Список забаненных пользователей:\n\n")

	for i, ban := range bans {
		name := "Неизвестный"
		if ban.FirstName != nil {
			name = *ban.FirstName
			if ban.LastName != nil {
				name += " " + *ban.LastName
			}
		}
		if ban.Username != nil {
			name += " (@" + *ban.Username + ")"
		}

		reason := "Не указана"
		if ban.Reason != nil {
			reason = *ban.Reason
		}

		response.WriteString(fmt.Sprintf("%d. ID: %d\n   Имя: %s\n   Причина: %s\n   Дата: %s\n\n",
			i+1, ban.UserID, name, reason, ban.BannedAt.Format("02.01.2006 15:04")))
	}

	utils.SendEphemeralReply(h.bot, message, response.String())
}

func (h *BotHandlers) handlePrivateMessage(message *tgbotapi.Message) {
	if h.banService.IsUserBanned(message.From.ID) {
		utils.SendReplyWithEffect(h.bot, message, "не хочу с тобой разговаривать", utils.EffectPoop)
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
	// Трекинг получения мема
	hasCaption := message.Caption != ""
	if message.MediaGroupID != "" {
		metrics.TrackMemeReceived("media_group", hasCaption)
	} else {
		metrics.TrackMemeReceived("photo", hasCaption)
	}

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
	// Трекинг получения видео
	hasCaption := message.Caption != ""
	metrics.TrackMemeReceived("video", hasCaption)

	firstName, lastName := utils.GetSenderName(message)
	caption := fmt.Sprintf("Мем прислал: %s %s", firstName, lastName)

	err := utils.SendVideoToChannel(h.bot, h.config.Channel, tgbotapi.FileID(message.Video.FileID), caption)
	if err != nil {
		log.Printf("Error sending video to channel: %v", err)
		metrics.TrackChannelPostError("video_send_failed")
		utils.SendReply(h.bot, message, "Произошла ошибка при отправке видео.")
		return
	}

	metrics.TrackMemePosted("video")
	services.RecordChannelMeme(message.Video.FileID, "video", message.From, "")
	utils.SendReplyWithEffect(h.bot, message, "Спасибо за мем! Пока-пока", utils.EffectFire)
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
		metrics.TrackChannelPostError("media_group_send_failed")
		utils.SendReply(h.bot, originalMessage, "Произошла ошибка при отправке медиа группы.")
		return
	}

	metrics.TrackMemePosted("media_group")
	for _, item := range group.Media {
		if photo, ok := item.(tgbotapi.InputMediaPhoto); ok {
			if fid, ok := photo.Media.(tgbotapi.FileID); ok {
				services.RecordChannelMeme(string(fid), "photo", originalMessage.From, groupID)
			}
		}
	}
	utils.SendReplyWithEffect(h.bot, originalMessage, "Спасибо за мем! Приходи еще", utils.EffectFire)
}

func (h *BotHandlers) sendSinglePhoto(fileID, caption string, message *tgbotapi.Message) {
	err := utils.SendToChannel(h.bot, h.config.Channel, tgbotapi.FileID(fileID), caption)
	if err != nil {
		log.Printf("Error sending photo to channel: %v", err)
		metrics.TrackChannelPostError("photo_send_failed")
		utils.SendReply(h.bot, message, "Произошла ошибка при отправке фото.")
		return
	}

	metrics.TrackMemePosted("photo")
	services.RecordChannelMeme(fileID, "photo", message.From, "")
	utils.SendReplyWithEffect(h.bot, message, "Спасибо за мем! Пока-пока", utils.EffectFire)
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

	// Берем caption из первого сообщения, если он есть
	caption := messages[0].Caption

	draft := utils.NewDraft(h.bot, messages[0])
	draft.Thinking()

	comment, err := h.openaiService.GenerateCommentFromImages(ctx, imageURLs, messages[0].Chat.ID, caption, draft.Update)
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

	// Учитываем подпись к фото, если она есть
	caption := message.Caption

	// Очень редко отвечаем на мем не текстом, а своим сгенерированным мемом
	if h.config.MemeImageProbability > 0 && rand.Float64() < h.config.MemeImageProbability {
		if h.tryRandomMemeReply(ctx, message, imageURL, caption) {
			return
		}
	}

	draft := utils.NewDraft(h.bot, message)
	draft.Thinking()

	comment, err := h.openaiService.GenerateCommentFromImage(ctx, imageURL, message.Chat.ID, caption, draft.Update)
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

	if memeID != "" && isMemeRemakeRequest(message.Text) {
		h.handleMemeRemakeRequest(ctx, message, memeID)
		return
	}

	draft := utils.NewDraft(h.bot, message)
	draft.Thinking()

	var response string

	if memeID != "" {
		metrics.TrackMemeInteraction()
		response, err = h.openaiService.GetMemeContextualResponse(ctx, message.Chat.ID, memeID, message.Text, draft.Update)
	} else {
		metrics.TrackDialogInteraction()
		response, err = h.openaiService.GetResponse(ctx, message.Text, message.Chat.ID, draft.Update)
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

var memeRemakeMarkers = []string{
	"как бы ты сделал",
	"как ты бы сделал",
	"как бы ты его сделал",
	"нарисуй",
	"сгенерируй",
	"сгенери",
	"свой вариант",
	"свою версию",
	"твоя версия",
	"твой вариант",
	"покажи как надо",
	"покажи, как надо",
	"сделай свой мем",
	"сделай мем",
	"переделай",
}

// isMemeRemakeRequest определяет, просит ли пользователь бота сделать свой вариант мема.
func isMemeRemakeRequest(text string) bool {
	t := strings.ToLower(text)
	for _, marker := range memeRemakeMarkers {
		if strings.Contains(t, marker) {
			return true
		}
	}
	return false
}

// extractMemeSource достаёт из истории мема исходные картинки и текстовый контекст обсуждения.
func extractMemeSource(history []models.MemeInteraction) (urls []string, context string) {
	var b strings.Builder
	for _, entry := range history {
		content := entry.Content
		switch {
		case strings.HasPrefix(content, "[MEME_IMAGE: "):
			urls = append(urls, strings.TrimSuffix(strings.TrimPrefix(content, "[MEME_IMAGE: "), "]"))
		case strings.HasPrefix(content, "[MEME_GROUP: "):
			list := strings.TrimSuffix(strings.TrimPrefix(content, "[MEME_GROUP: "), "]")
			urls = append(urls, strings.Split(list, ", ")...)
		case strings.HasPrefix(content, "[GENERATED_MEME]"):
			b.WriteString("Сталин (ответил своим мемом): " + strings.TrimSpace(strings.TrimPrefix(content, "[GENERATED_MEME]")) + "\n")
		default:
			role := "Пользователь"
			if entry.Role == "assistant" {
				role = "Сталин"
			}
			b.WriteString(role + ": " + content + "\n")
		}
	}
	return urls, b.String()
}

// handleMemeRemakeRequest — пользователь попросил бота показать свой вариант мема.
func (h *BotHandlers) handleMemeRemakeRequest(ctx context.Context, message *tgbotapi.Message, memeID string) {
	metrics.TrackMemeInteraction()

	history, err := h.openaiService.GetMemeHistory(message.Chat.ID, memeID)
	if err != nil {
		log.Printf("Error getting meme history for remake: %v", err)
		utils.SendReply(h.bot, message, "Не удалось вспомнить этот мем. Попробуйте позже.")
		return
	}

	sourceURLs, memeContext := extractMemeSource(history)

	request := fmt.Sprintf("Пользователь просит показать, как бы ТЫ сделал этот мем. Его слова: \"%s\". Придумай свою версию этого мема — сохрани тему, но сделай смешнее и в своём стиле.", message.Text)

	stopAction := utils.KeepChatAction(h.bot, message, tgbotapi.ChatUploadPhoto)
	imageData, caption, err := h.openaiService.GenerateMemeRemake(ctx, sourceURLs, memeContext, request, "request")
	stopAction()
	if err != nil {
		log.Printf("Error generating meme remake: %v", err)
		response, err := h.openaiService.GetMemeContextualResponse(ctx, message.Chat.ID, memeID, message.Text, nil)
		if err != nil {
			log.Printf("Error getting fallback response: %v", err)
			utils.SendReply(h.bot, message, "Что-то пошло не так, даже мем не нарисовался. Попробуйте позже.")
			return
		}
		if sentMessage, err := utils.SendReply(h.bot, message, response); err == nil && sentMessage != nil {
			h.openaiService.AddCommentMapping(sentMessage.MessageID, memeID)
		}
		return
	}

	sentMessage, err := utils.SendPhotoReply(h.bot, message, imageData, caption)
	if err != nil {
		log.Printf("Error sending meme remake: %v", err)
		return
	}

	h.openaiService.AddMemeInteraction(message.Chat.ID, memeID, "user", message.Text)
	h.openaiService.AddMemeInteraction(message.Chat.ID, memeID, "assistant", "[GENERATED_MEME] "+caption)

	if sentMessage != nil {
		h.openaiService.AddCommentMapping(sentMessage.MessageID, memeID)
	}
}

// tryRandomMemeReply с небольшим шансом отвечает на мем не текстом, а своим сгенерированным мемом.
// Возвращает true, если мем-ответ отправлен.
func (h *BotHandlers) tryRandomMemeReply(ctx context.Context, message *tgbotapi.Message, imageURL, userCaption string) bool {
	request := "Прокомментируй этот мем не словами, а СВОИМ мемом-ответом: придумай мем, который станет панчлайном или подколом к присланному мему."
	if userCaption != "" {
		request += fmt.Sprintf(" Автор подписал свой мем: \"%s\".", userCaption)
	}

	stopAction := utils.KeepChatAction(h.bot, message, tgbotapi.ChatUploadPhoto)
	imageData, caption, err := h.openaiService.GenerateMemeRemake(ctx, []string{imageURL}, "", request, "random")
	stopAction()
	if err != nil {
		log.Printf("Error generating random meme reply: %v", err)
		return false
	}

	sentMessage, err := utils.SendPhotoReply(h.bot, message, imageData, caption)
	if err != nil {
		log.Printf("Error sending random meme reply: %v", err)
		return false
	}

	memeID := message.Photo[len(message.Photo)-1].FileID
	h.openaiService.AddMemeInteraction(message.Chat.ID, memeID, "user", fmt.Sprintf("[MEME_IMAGE: %s]", imageURL))
	h.openaiService.AddMemeInteraction(message.Chat.ID, memeID, "assistant", "[GENERATED_MEME] "+caption)

	if sentMessage != nil {
		h.openaiService.AddCommentMapping(sentMessage.MessageID, memeID)
	}

	log.Printf("Replied to meme with generated meme: %s", caption)
	return true
}

func (h *BotHandlers) sendInvalidChatMessage(message *tgbotapi.Message) {
	text := "Хорошая попытка, но я сделан только для паблика @stalinfollower"
	utils.SendReply(h.bot, message, text)
}
