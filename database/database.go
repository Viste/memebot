package database

import (
	"log"
	"memebot/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(databaseURL string) error {
	var err error

	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	DB, err = gorm.Open(postgres.Open(databaseURL), config)
	if err != nil {
		return err
	}

	log.Println("Connected to PostgreSQL database")
	return nil
}

func Migrate() error {
	err := DB.AutoMigrate(
		&models.Admin{},
		&models.BannedUser{},
		&models.Meme{},
		&models.MemeComment{},
		&models.MemeHistory{},
		&models.UserDialog{},
		&models.MemeInteraction{},
		&models.CommentMemeMapping{},
	)

	if err != nil {
		return err
	}

	log.Println("Database migration completed")
	return nil
}

func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
