package app

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"hookah-bot/internal/delivery/telegram"
	"hookah-bot/internal/repository/memory"
	"hookah-bot/internal/service"

	tele "gopkg.in/telebot.v3"
)

func Run(token string, adminID int64) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	repo := memory.NewBookingRepo()
	bookingService := service.NewBookingService(repo)
	handlers := telegram.NewHandlers(bookingService, adminID)
	handlers.InitRoutes(b)

	log.Printf("Бот @%s успешно запущен! Admin ID: %d", b.Me.Username, adminID)

	// --- ЗАПУСК ВЕБ-СЕРВЕРА ---
	go func() {
		http.HandleFunc("/api/book", func(w http.ResponseWriter, r *http.Request) {
			// ДОБАВЛЕНО ПОЛЕ Time В СТРУКТУРУ
			var data struct {
				Table  string `json:"table"`
				Time   string `json:"time"`
				UserID int64  `json:"userId"`
			}

			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// ОБНОВЛЕН ЛОГ В ТЕРМИНАЛЕ
			log.Printf("🔥 НОВАЯ БРОНЬ! Пользователь %d выбрал %s на %s\n", data.UserID, data.Table, data.Time)

			// ОБНОВЛЕН ТЕКСТ СООБЩЕНИЯ (добавлено время)
			user := &tele.User{ID: data.UserID}
			msg := "✅ Спасибо! Вы успешно забронировали: " + data.Table + " на " + data.Time

			_, err := b.Send(user, msg)
			if err != nil {
				log.Printf("Ошибка при отправке сообщения: %v", err)
			}

			w.WriteHeader(http.StatusOK)
		})

		fs := http.FileServer(http.Dir("./webapp"))
		http.Handle("/", fs)

		log.Println("Локальный сервер Web App запущен на http://localhost:8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal("Ошибка запуска сервера: ", err)
		}
	}()
	// --------------------------

	b.Start()
}
