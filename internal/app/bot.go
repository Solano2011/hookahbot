package app

import (
	"encoding/json"
	"html/template"
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

		// 1. Отдаем статические файлы (картинки) по пути /img/
		http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("./webapp/img"))))

		// 2. Обрабатываем главную страницу через шаблонизатор
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Защита от лишних запросов
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}

			// Парсим все шаблоны при каждом запросе страницы.
			// Это позволяет изменять HTML-файлы без перезапуска Go-сервера.
			tmpl, err := template.ParseGlob("webapp/templates/*.html")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Рендерим главный шаблон (base.html)
			if err := tmpl.ExecuteTemplate(w, "base.html", nil); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		log.Println("Локальный сервер Web App запущен на http://localhost:8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal("Ошибка запуска сервера: ", err)
		}
	}()
	// --------------------------

	b.Start()
}
