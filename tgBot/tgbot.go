package tgbot

import (
	"fileConvertor/utils"
	"io"
	"net/http"
	"os"
	"path"

	fileprocessor "fileConvertor/fileProcessor"

	"github.com/defskela/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func StartBot() {

	err := godotenv.Load()
	if err != nil {
		logger.Error(err)
		return
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
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
			logger.Info("Получено сообщение", update.Message.Text, "от", update.Message.From.FirstName, update.Message.From.LastName)
			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					handleStart(bot, update.Message)
				case "convert":
					handleConvert(bot, update.Message)
				default:
					handleUnknownCommand(bot, update.Message)
				}
			} else if update.Message.Document != nil {
				handleConvert(bot, update.Message)
			} else {
				handleUnknownCommand(bot, update.Message)
			}
		}
	}
}

func handleStart(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, utils.WelcomeText)

	if _, err := bot.Send(msg); err != nil {
		logger.Debug("Не получилось отправить сообщение /start: %v", err)
	}
}

func handleUnknownCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, utils.ReplyText)

	if _, err := bot.Send(msg); err != nil {
		logger.Warn("Не получилось отправить сообщение о неизвестной команде: %v", err)
	}
}

func handleConvert(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if message.Document != nil {
		// Проверка на тип файла. Нужен только pdf или word
		if !(message.Document.MimeType == "application/pdf" || message.Document.MimeType == "application/msword") {
			msg := tgbotapi.NewMessage(message.Chat.ID, utils.SendFileText)
			if _, err := bot.Send(msg); err != nil {
				logger.Warn("Не получилось отправить сообщение о неизвестном типе файла %v", err)
			}
			return
		}
		fileID := message.Document.FileID
		file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
		if err != nil {
			logger.Debug("Не получилось принять файл: %v", err)
			return
		}

		fileURL := file.Link(bot.Token)
		logger.Info("Получен файл: %v", fileURL)

		response, err := http.Get(fileURL)

		if err != nil {
			logger.Debug("Не получилось скачать файл: %v", err)
			return
		}

		defer response.Body.Close()
		filePath := path.Join("files", message.Document.FileName)

		logger.Debug("PATH", filePath)
		out, err := os.Create(filePath)
		if err != nil {
			logger.Debug("Не получилось создать файл:", err)
			return
		}
		defer out.Close()

		_, err = io.Copy(out, response.Body)
		if err != nil {
			logger.Debug("Не получилось сохранить файл: %v", err)
			return
		}

		msg := tgbotapi.NewMessage(message.Chat.ID, utils.ReceivingFileText)
		if _, err := bot.Send(msg); err != nil {
			logger.Warn("Не получилось отправить сообщение о получении файла %v", err)
		}

		if message.Document.MimeType == "application/pdf" {
			logger.Info("Файл PDF")
			fileprocessor.ConvertPDFToWord(message.Document.FileName)
		} else if message.Document.MimeType == "application/msword" {
			logger.Info("Файл Word")
			fileprocessor.ConvertWordToPDF(message.Document.FileName)
		}

		resultFile, err := os.Open("files/output.docx")
		if err != nil {
			logger.Warn("Failed to open file: %v", err)
		}
		defer resultFile.Close()

		doc := tgbotapi.NewDocument(message.Chat.ID, tgbotapi.FileReader{
			Name:   "output.docx",
			Reader: resultFile,
		})

		_, err = bot.Send(doc)
		if err != nil {
			logger.Warn("Failed to send document: %v", err)
		}

	} else {
		msg := tgbotapi.NewMessage(message.Chat.ID, utils.SendFileText)
		if _, err := bot.Send(msg); err != nil {
			logger.Warn("Не получилось отправить сообщение об ожидании файла %v", err)
		}
	}

}
