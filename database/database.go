package database

import (
	"log"
	"memebot/models"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(databaseURL string) error {
	var err error

	connConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return err
	}
=
	connConfig.DefaultQueryExecMode = pgx.QueryExecModeExec

	sqlDB := stdlib.OpenDB(*connConfig)

	config := &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Info),
		PrepareStmt: false,
		QueryFields: true,
	}

	DB, err = gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), config)
	if err != nil {
		return err
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

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
