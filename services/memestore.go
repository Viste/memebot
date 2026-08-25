package services

import (
	"log"
	"time"

	"memebot/database"
	"memebot/models"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"gorm.io/gorm/clause"
)

// RecordChannelMeme сохраняет мем, отправленный ботом в канал, в таблицу memes.
func RecordChannelMeme(fileID, fileType string, from *tgbotapi.User, mediaGroupID string) {
	if fileID == "" {
		return
	}

	meme := models.Meme{
		FileID:             fileID,
		FileType:           fileType,
		ForwardedToChannel: true,
		CreatedAt:          time.Now(),
	}

	if from != nil {
		meme.UserID = from.ID
		if from.UserName != "" {
			meme.Username = stringPtr(from.UserName)
		}
		if from.FirstName != "" {
			meme.FirstName = stringPtr(from.FirstName)
		}
		if from.LastName != "" {
			meme.LastName = stringPtr(from.LastName)
		}
	}

	if mediaGroupID != "" {
		meme.MediaGroupID = stringPtr(mediaGroupID)
	}

	err := database.DB.
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "file_id"}}, DoNothing: true}).
		Create(&meme).Error
	if err != nil {
		log.Printf("RecordChannelMeme error for %s: %v", fileID, err)
	}
}

// RecentChannelPhotoIDs возвращает file_id фото-мемов, отправленных в канал с момента since.
func RecentChannelPhotoIDs(since time.Time) []string {
	var ids []string
	err := database.DB.Model(&models.Meme{}).
		Where("created_at >= ? AND file_type = ? AND forwarded_to_channel = ?", since, "photo", true).
		Order("created_at DESC").
		Pluck("file_id", &ids).Error
	if err != nil {
		log.Printf("RecentChannelPhotoIDs error: %v", err)
	}
	return ids
}
