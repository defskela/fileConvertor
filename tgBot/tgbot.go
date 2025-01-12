package main

import (
	"fileConvertor/utils"
	"os"

	"github.com/defskela/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		logger.Error(err)
		return
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		logger.Error(err)
		return
	}

	// bot.Debug = true
	logger.Info("Бот начал работу")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			switch update.Message.Text {
			case "/start":
				handleStart(bot, update.Message)
			default:
				handleUnknownCommand(bot, update.Message)
			}
		}
	}
}

func handleStart(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	welcomeText := utils.WelcomeText

	msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)

	if _, err := bot.Send(msg); err != nil {
		logger.Debug("Error sending /start message: %v", err)
	}
}

func handleUnknownCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	replyText := utils.ReplyText

	msg := tgbotapi.NewMessage(message.Chat.ID, replyText)

	if _, err := bot.Send(msg); err != nil {
		logger.Debug("Error sending unknown command message: %v", err)
	}
}
