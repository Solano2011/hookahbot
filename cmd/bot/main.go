package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"hookah-bot/internal/app"
	"hookah-bot/internal/repository/postgres"

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

	// Строим строку подключения из переменных окружения
	dbHost := getEnvOrDefault("DB_HOST", "localhost")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "hookah_user")
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		log.Fatal("Укажите DB_PASSWORD в переменных окружения")
	}
	dbName := getEnvOrDefault("DB_NAME", "hookah_db")
	dbSSLMode := getEnvOrDefault("DB_SSLMODE", "require")

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode)

	db, err := postgres.NewPostgresDB(connString)
	if err != nil {
		log.Fatalf("Не удалось инициализировать базу данных: %v", err)
	}
	// Закрываем соединение при остановке бота
	defer db.Conn.Close(context.Background())

	// Передаем db внутрь app.Run
	app.Run(token, adminID, db)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
