package main

import (
	"log"
	"memebot/config"
	"memebot/database"
	"memebot/handlers"
	"memebot/services"
	"memebot/web"
	"os"
	"os/signal"
	"sync"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	if err := config.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := database.Connect(config.AppConfig.DatabaseURL); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(config.AppConfig.Token)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	bot.Debug = false
	log.Printf("Bot authorized as %s", bot.Self.UserName)

	openaiService := services.NewOpenAIService(
		config.AppConfig.OpenAIAPIKey,
		config.AppConfig.OpenAIBaseURL,
	)

	botHandlers := handlers.NewBotHandlers(bot, openaiService, config.AppConfig)
	webServer := web.NewServer(config.AppConfig.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting web server on port %s", config.AppConfig.Port)
		if err := webServer.Start(); err != nil {
			log.Printf("Web server error: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runBot(bot, botHandlers)
	}()

	<-quit
	log.Println("Shutting down...")

	if err := database.Close(); err != nil {
		log.Printf("Error closing database connection: %v", err)
	}

	bot.StopReceivingUpdates()

	log.Println("Bot stopped!")
}

func runBot(bot *tgbotapi.BotAPI, handlers *handlers.BotHandlers) {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := bot.GetUpdatesChan(updateConfig)

	log.Println("Starting bot polling...")

	for update := range updates {
		go func(update tgbotapi.Update) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in update handler: %v", r)
				}
			}()

			handlers.HandleUpdate(update)
		}(update)
	}
}
