package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Общие метрики бота
	MessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "memebot_messages_received_total",
			Help: "Общее количество полученных сообщений",
		},
		[]string{"type", "chat_type"}, // type: text/photo/video, chat_type: private/group
	)

	CommandsExecuted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "memebot_commands_executed_total",
			Help: "Общее количество выполненных команд",
		},
		[]string{"command"},
	)

	// Метрики мемов
	MemesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "memebot_memes_received_total",
			Help: "Общее количество полученных мемов",
		},
		[]string{"type"}, // type: photo/video/media_group
	)

	MemesPostedToChannel = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "memebot_memes_posted_total",
			Help: "Общее количество мемов, отправленных в канал",
		},
		[]string{"type"}, // type: photo/video/media_group
	)

	// Метрики комментариев (AI)
	CommentsGenerated = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "memebot_comments_generated_total",
			Help: "Общее количество сгенерированных комментариев",
		},
	)

	CommentGenerationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "memebot_comment_generation_duration_seconds",
			Help:    "Время генерации комментария в секундах",
			Buckets: prometheus.DefBuckets, // [.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10]
		},
	)

	CommentGenerationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "memebot_comment_generation_errors_total",
			Help: "Количество ошибок при генерации комментариев",
		},
		[]string{"error_type"},
	)

	// Метрики OpenAI
	OpenAIRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "memebot_openai_requests_total",
			Help: "Общее количество запросов к OpenAI API",
		},
		[]string{"model", "type"}, // type: comment/response/contextual
	)

	OpenAIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "memebot_openai_request_duration_seconds",
			Help:    "Время выполнения запроса к OpenAI в секундах",
			Buckets: []float64{.1, .25, .5, 1, 2, 5, 10, 20, 30},
		},
		[]string{"model", "type"},
	)

	OpenAIErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "memebot_openai_errors_total",
			Help: "Количество ошибок при обращении к OpenAI",
		},
		[]string{"error_type"},
	)

	// Метрики токенов OpenAI
	OpenAITokensUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "memebot_openai_tokens_used_total",
			Help: "Общее количество использованных токенов OpenAI",
		},
		[]string{"model", "type"}, // type: prompt/completion
	)

	// Метрики банов
	UsersBanned = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "memebot_users_banned_total",
			Help: "Общее количество забаненных пользователей",
		},
	)

	UsersUnbanned = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "memebot_users_unbanned_total",
			Help: "Общее количество разбаненных пользователей",
		},
	)

	BannedUsersActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "memebot_banned_users_active",
			Help: "Текущее количество активных банов",
		},
	)

	// Метрики диалогов
	DialogInteractions = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "memebot_dialog_interactions_total",
			Help: "Общее количество взаимодействий в диалогах",
		},
	)

	MemeInteractions = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "memebot_meme_interactions_total",
			Help: "Общее количество взаимодействий с мемами (ответы на комментарии)",
		},
	)

	// Метрики базы данных
	DatabaseOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "memebot_database_operations_total",
			Help: "Общее количество операций с базой данных",
		},
		[]string{"operation", "table"}, // operation: create/read/update/delete
	)

	DatabaseErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "memebot_database_errors_total",
			Help: "Количество ошибок при работе с базой данных",
		},
		[]string{"operation", "table"},
	)

	// Метрики каналов
	ChannelPostsSuccessful = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "memebot_channel_posts_successful_total",
			Help: "Количество успешных постов в канал",
		},
	)

	ChannelPostsErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "memebot_channel_posts_errors_total",
			Help: "Количество ошибок при постинге в канал",
		},
		[]string{"error_type"},
	)

	// Метрики личности Сталина (для аналитики)
	StalinPersonaUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "memebot_stalin_persona_used_total",
			Help: "Количество использований каждой личности Сталина",
		},
		[]string{"persona"}, // persona: strict/ironic/relaxed
	)

	// Метрики с подписями к мемам
	MemesWithCaption = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "memebot_memes_with_caption_total",
			Help: "Количество мемов с подписями",
		},
	)

	MemesWithoutCaption = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "memebot_memes_without_caption_total",
			Help: "Количество мемов без подписей",
		},
	)
)

// Helper функции для удобства использования

// TrackMessageReceived отслеживает полученное сообщение
func TrackMessageReceived(messageType, chatType string) {
	MessagesReceived.WithLabelValues(messageType, chatType).Inc()
}

// TrackCommandExecuted отслеживает выполнение команды
func TrackCommandExecuted(command string) {
	CommandsExecuted.WithLabelValues(command).Inc()
}

// TrackMemeReceived отслеживает получение мема
func TrackMemeReceived(memeType string, hasCaption bool) {
	MemesReceived.WithLabelValues(memeType).Inc()
	if hasCaption {
		MemesWithCaption.Inc()
	} else {
		MemesWithoutCaption.Inc()
	}
}

// TrackMemePosted отслеживает отправку мема в канал
func TrackMemePosted(memeType string) {
	MemesPostedToChannel.WithLabelValues(memeType).Inc()
	ChannelPostsSuccessful.Inc()
}

// TrackChannelPostError отслеживает ошибку при постинге в канал
func TrackChannelPostError(errorType string) {
	ChannelPostsErrors.WithLabelValues(errorType).Inc()
}

// TrackCommentGenerated отслеживает успешную генерацию комментария
func TrackCommentGenerated(duration float64) {
	CommentsGenerated.Inc()
	CommentGenerationDuration.Observe(duration)
}

// TrackCommentError отслеживает ошибку при генерации комментария
func TrackCommentError(errorType string) {
	CommentGenerationErrors.WithLabelValues(errorType).Inc()
}

// TrackOpenAIRequest отслеживает запрос к OpenAI
func TrackOpenAIRequest(model, requestType string, duration float64) {
	OpenAIRequests.WithLabelValues(model, requestType).Inc()
	OpenAIRequestDuration.WithLabelValues(model, requestType).Observe(duration)
}

// TrackOpenAIError отслеживает ошибку OpenAI
func TrackOpenAIError(errorType string) {
	OpenAIErrors.WithLabelValues(errorType).Inc()
}

// TrackOpenAITokens отслеживает использование токенов
func TrackOpenAITokens(model, tokenType string, count int) {
	OpenAITokensUsed.WithLabelValues(model, tokenType).Add(float64(count))
}

// TrackUserBanned отслеживает бан пользователя
func TrackUserBanned() {
	UsersBanned.Inc()
	BannedUsersActive.Inc()
}

// TrackUserUnbanned отслеживает разбан пользователя
func TrackUserUnbanned() {
	UsersUnbanned.Inc()
	BannedUsersActive.Dec()
}

// SetActiveBans устанавливает текущее количество активных банов
func SetActiveBans(count int) {
	BannedUsersActive.Set(float64(count))
}

// TrackDatabaseOperation отслеживает операцию с БД
func TrackDatabaseOperation(operation, table string) {
	DatabaseOperations.WithLabelValues(operation, table).Inc()
}

// TrackDatabaseError отслеживает ошибку БД
func TrackDatabaseError(operation, table string) {
	DatabaseErrors.WithLabelValues(operation, table).Inc()
}

// TrackDialogInteraction отслеживает взаимодействие в диалоге
func TrackDialogInteraction() {
	DialogInteractions.Inc()
}

// TrackMemeInteraction отслеживает взаимодействие с мемом
func TrackMemeInteraction() {
	MemeInteractions.Inc()
}

// TrackStalinPersona отслеживает использование личности Сталина
func TrackStalinPersona(personaIndex int) {
	var personaName string
	switch personaIndex {
	case 0:
		personaName = "strict_modern"
	case 1:
		personaName = "ironic_memologist"
	case 2:
		personaName = "relaxed_grandpa"
	default:
		personaName = "unknown"
	}
	StalinPersonaUsed.WithLabelValues(personaName).Inc()
}
