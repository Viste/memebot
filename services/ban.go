package services

import (
	"fmt"
	"log"
	"memebot/database"
	"memebot/models"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"gorm.io/gorm"
)

type BanService struct {
	bot *tgbotapi.BotAPI
}

func NewBanService(bot *tgbotapi.BotAPI) *BanService {
	return &BanService{
		bot: bot,
	}
}

// MigrateConfigBans мигрирует баны из конфига в базу данных
func (bs *BanService) MigrateConfigBans(bannedUserIDs []int64) error {
	log.Printf("Migrating %d banned users from config to database", len(bannedUserIDs))

	for _, userID := range bannedUserIDs {
		var existingBan models.BannedUser
		err := database.DB.Where("user_id = ?", userID).First(&existingBan).Error

		if err == gorm.ErrRecordNotFound {
			ban := models.BannedUser{
				UserID:   userID,
				BannedAt: time.Now(),
				Reason:   stringPtr("Migrated from config"),
			}

			if userInfo, err := bs.getUserInfo(userID); err == nil {
				ban.Username = userInfo.Username
				ban.FirstName = userInfo.FirstName
				ban.LastName = userInfo.LastName
			}

			if err := database.DB.Create(&ban).Error; err != nil {
				log.Printf("Error migrating ban for user %d: %v", userID, err)
				continue
			}

			log.Printf("Migrated ban for user %d", userID)
		} else if err != nil {
			log.Printf("Error checking existing ban for user %d: %v", userID, err)
		} else {
			log.Printf("User %d already banned in database", userID)
		}
	}

	return nil
}

// IsUserBanned проверяет бан пользователя в БД
func (bs *BanService) IsUserBanned(userID int64) bool {
	var ban models.BannedUser
	err := database.DB.Where("user_id = ?", userID).First(&ban).Error
	return err == nil
}

// BanUser банит пользователя
func (bs *BanService) BanUser(userID int64, bannedBy int64, reason string) error {
	if bs.IsUserBanned(userID) {
		return fmt.Errorf("user %d is already banned", userID)
	}

	ban := models.BannedUser{
		UserID:   userID,
		BannedAt: time.Now(),
		BannedBy: &bannedBy,
		Reason:   &reason,
	}

	// Получаем информацию о пользователе
	if userInfo, err := bs.getUserInfo(userID); err == nil {
		ban.Username = userInfo.Username
		ban.FirstName = userInfo.FirstName
		ban.LastName = userInfo.LastName
	}

	return database.DB.Create(&ban).Error
}

// UnbanUser разбанивает пользователя
func (bs *BanService) UnbanUser(userID int64) error {
	result := database.DB.Where("user_id = ?", userID).Delete(&models.BannedUser{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user %d is not banned", userID)
	}

	return nil
}

// GetBannedUsers возвращает список забаненных пользователей
func (bs *BanService) GetBannedUsers(limit int) ([]models.BannedUser, error) {
	var bans []models.BannedUser
	err := database.DB.Order("banned_at DESC").Limit(limit).Find(&bans).Error
	return bans, err
}

// getUserInfo получает информацию о пользователе из Telegram
func (bs *BanService) getUserInfo(userID int64) (*UserInfo, error) {
	// пока возвращаем nil, информация будет заполняться при следующем сообщении
	return nil, fmt.Errorf("cannot get user info from Telegram API")
}

type UserInfo struct {
	Username  *string
	FirstName *string
	LastName  *string
}

// stringPtr возвращает указатель на строку
func stringPtr(s string) *string {
	return &s
}
