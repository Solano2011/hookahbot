package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"hookah-bot/internal/app"
	"hookah-bot/internal/repository/postgres" // Добавили импорт твоего пакета

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

	// --- НОВЫЙ БЛОК: Подключение к базе данных ---

	// ВАЖНО: Замени "твой_пароль" на реальный пароль от БД!
	// В будущем мы тоже вынесем эту строку в .env файл.
	connString := "postgres://hookah_user:Solano2011!.-@localhost:5432/hookah_db?sslmode=disable"
	db, err := postgres.NewPostgresDB(connString)
	if err != nil {
		log.Fatalf("Не удалось инициализировать базу данных: %v", err)
	}
	// Закрываем соединение при остановке бота
	defer db.Conn.Close(context.Background())

	// --- КОНЕЦ НОВОГО БЛОКА ---

	// Передаем db внутрь app.Run (нам придется немного изменить app.Run)
	app.Run(token, adminID, db)
}
