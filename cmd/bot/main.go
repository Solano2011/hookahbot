package main

import (
	"hookah-bot/internal/app"
	"log"
	"os"
	"strconv"
)

func main() {
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
