package main

import (
	"log"
	"os"
	"strconv"

	"hookah-bot/internal/app"

	"github.com/joho/godotenv"
)

func main() {
	// Загружаем переменные из файла .env
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используем системные переменные")
	}

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("Укажите BOT_TOKEN в переменных окружения")
	}

	adminIDStr := os.Getenv("ADMIN_ID")
	var adminID int64
	if adminIDStr != "" {
		parsed, err := strconv.ParseInt(adminIDStr, 10, 64)
		if err != nil {
			log.Printf("Некорректный ADMIN_ID: %v", err)
		} else {
			adminID = parsed
		}
	}

	app.Run(token, adminID)
}
