package config

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Token         string  `json:"token"`
	Channel       string  `json:"channel"`
	BannedUserIDs []int64 `json:"banned_user_ids"`
	AdminIDs      []int64 `json:"admin_ids"`

	// Database
	DatabaseURL string

	// OpenAI
	OpenAIAPIKey  string
	OpenAIBaseURL string

	// Web server
	Port string
}

var AppConfig *Config

func Load() error {
	// Загружаем .env файл если он существует
	_ = godotenv.Load()

	config := &Config{}

	if data, err := os.ReadFile("config.json"); err == nil {
		if err := json.Unmarshal(data, config); err != nil {
			log.Printf("Warning: failed to parse config.json: %v", err)
		}
	}

	if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
		config.Token = token
	}

	if channel := os.Getenv("TELEGRAM_CHANNEL"); channel != "" {
		config.Channel = channel
	}

	if bannedUsers := os.Getenv("BANNED_USER_IDS"); bannedUsers != "" {
		userIDs := strings.Split(bannedUsers, ",")
		config.BannedUserIDs = make([]int64, 0, len(userIDs))
		for _, idStr := range userIDs {
			if id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64); err == nil {
				config.BannedUserIDs = append(config.BannedUserIDs, id)
			}
		}
	}

	if adminUsers := os.Getenv("ADMIN_USER_IDS"); adminUsers != "" {
		userIDs := strings.Split(adminUsers, ",")
		config.AdminIDs = make([]int64, 0, len(userIDs))
		for _, idStr := range userIDs {
			if id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64); err == nil {
				config.AdminIDs = append(config.AdminIDs, id)
			}
		}
	}

	config.DatabaseURL = os.Getenv("DATABASE_URL")
	if config.DatabaseURL == "" {
		config.DatabaseURL = "postgres://user:password@localhost/meme_bot?sslmode=disable"
	}

	config.OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
	config.OpenAIBaseURL = os.Getenv("OPENAI_BASE_URL")
	if config.OpenAIBaseURL == "" {
		config.OpenAIBaseURL = "http://31.172.78.152:9000/v1"
	}

	config.Port = os.Getenv("PORT")
	if config.Port == "" {
		config.Port = "8080"
	}

	AppConfig = config
	return nil
}

func (c *Config) IsUserBanned(userID int64) bool {
	for _, bannedID := range c.BannedUserIDs {
		if bannedID == userID {
			return true
		}
	}
	return false
}

func (c *Config) IsUserAdmin(userID int64) bool {
	for _, adminID := range c.AdminIDs {
		if adminID == userID {
			return true
		}
	}
	return false
}
