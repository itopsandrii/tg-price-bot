package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Вынесем токен в константу. В реальном проекте лучше использовать переменные окружения.
const (
	botToken      = "797187266Э" // <-- ЗАМЕНИТЕ ВАШИМ ТОКЕНОМ
	imagesDir     = "images"
	welcomeMsg    = "Привет! 👋 Я бот для поиска цен предметов по фотографии. Пришли мне фото, и я скажу тебе цену!"
	unknownCmdMsg = "Неизвестная команда 😕"
	sendPhotoMsg  = "Отправь мне фото, и я постараюсь узнать его цену!"
	photoSavedMsg = "Изображение получено и сохранено! ✅"
)

func main() {
	// Создаём папку для сохранения изображений
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		log.Panic("Не удалось создать папку 'images':", err)
	}

	// Инициализируем бота
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic("Ошибка инициализации бота:", err)
	}

	bot.Debug = true
	log.Printf("Авторизован как %s", bot.Self.UserName)

	// Настраиваем получение обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Запускаем обработку обновлений в отдельной функции
	handleUpdates(bot, updates)
}

// handleUpdates обрабатывает все входящие обновления от Telegram
func handleUpdates(bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		if update.Message.IsCommand() {
			handleCommand(bot, update.Message)
		} else if len(update.Message.Photo) > 0 {
			handlePhoto(bot, update.Message)
		} else {
			// Ответ на обычные текстовые сообщения
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, sendPhotoMsg)
			bot.Send(msg)
		}
	}
}

// handleCommand обрабатывает команды
func handleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	var responseText string
	switch message.Command() {
	case "start":
		responseText = welcomeMsg
	default:
		responseText = unknownCmdMsg
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	bot.Send(msg)
}

// handlePhoto обрабатывает сообщения с фотографиями
func handlePhoto(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// Берем фотографию лучшего качества (последнюю в массиве)
	fileID := message.Photo[len(message.Photo)-1].FileID

	// Получаем информацию о файле (включая FilePath)
	fileConfig := tgbotapi.FileConfig{FileID: fileID}
	file, err := bot.GetFile(fileConfig)
	if err != nil {
		log.Printf("Ошибка получения файла: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Не удалось получить файл. 😢"))
		return
	}
	log.Printf("Информация о файле получена: %+v", file)

	// Скачиваем файл по полученному FilePath
	err = downloadFile(bot.Token, file.FilePath)
	if err != nil {
		log.Printf("Ошибка скачивания файла: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Не удалось скачать файл. 😢"))
		return
	}

	// Отправляем подтверждение пользователю
	msg := tgbotapi.NewMessage(message.Chat.ID, photoSavedMsg)
	bot.Send(msg)
}

// downloadFile скачивает файл по URL и сохраняет его
func downloadFile(token, filePath string) error {
	// Формируем URL для скачивания файла
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", token, filePath)

	// Выполняем GET-запрос для скачивания файла
	response, err := http.Get(fileURL)
	if err != nil {
		return fmt.Errorf("ошибка при выполнении http.Get: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("неверный статус-код ответа: %s", response.Status)
	}

	// Создаем уникальное имя файла на основе его пути в Telegram
	fileName := filepath.Base(filePath)
	fullPath := filepath.Join(imagesDir, fileName)

	// Создаем файл на диске
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("ошибка создания файла: %w", err)
	}
	defer file.Close()

	// Копируем содержимое ответа (файл) в созданный файл
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return fmt.Errorf("ошибка копирования содержимого файла: %w", err)
	}

	log.Printf("Файл успешно скачан и сохранен: %s", fullPath)
	return nil
}
