package app

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"hookah-bot/internal/delivery/telegram"
	"hookah-bot/internal/repository/postgres"
	"hookah-bot/internal/service"

	tele "gopkg.in/telebot.v3"
)

// Добавили db *postgres.DB третьим параметром
func Run(token string, adminID int64, db *postgres.DB) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	repo := postgres.NewBookingRepo(db)

	bookingService := service.NewBookingService(repo)
	handlers := telegram.NewHandlers(bookingService, adminID)
	handlers.InitRoutes(b)

	log.Printf("Бот @%s успешно запущен! Admin ID: %d", b.Me.Username, adminID)

	// --- ЗАПУСК ВЕБ-СЕРВЕРА ---
	go func() {
		http.HandleFunc("/api/book", func(w http.ResponseWriter, r *http.Request) {
			var data struct {
				Table  string `json:"table"`
				Time   string `json:"time"`
				UserID int64  `json:"userId"`
				Name   string `json:"name"`  // Принимаем имя с фронтенда
				Phone  string `json:"phone"` // Принимаем телефон с фронтенда
			}

			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			ctx := context.Background()
			// 0. СОЗДАЕМ ЧЕРНОВИК НА ЛЕТУ
			if err := bookingService.StartBookingDraft(ctx, data.UserID, "Общий лаунж"); err != nil {
				log.Printf("Ошибка создания черновика: %v", err)
				http.Error(w, "Ошибка создания черновика", http.StatusInternalServerError)
				return
			}

			// 1. СОХРАНЯЕМ СТОЛ В СЕРВИС
			if err := bookingService.SetBookingTable(ctx, data.UserID, data.Table); err != nil {
				log.Printf("Ошибка сохранения стола: %v", err)
				http.Error(w, "Ошибка сохранения стола", http.StatusInternalServerError)
				return
			}

			// 2. ФИНАЛИЗИРУЕМ БРОНЬ (сохраняем время)
			booking, err := bookingService.CompleteBookingDraft(ctx, data.UserID, data.Time, data.Name, data.Phone)
			if err != nil {
				log.Printf("Ошибка завершения брони: %v", err)
				http.Error(w, "Ошибка завершения брони", http.StatusConflict)
				return
			}

			// Записываем имя и телефон в структуру брони
			booking.UserName = data.Name
			booking.Phone = data.Phone

			log.Printf("🔥 НОВАЯ БРОНЬ! Пользователь %s (%s) выбрал %s на %s\n", data.Name, data.Phone, booking.Table, booking.TimeSlot)

			// 3. Отправляем красивое сообщение пользователю
			user := &tele.User{ID: data.UserID}
			text := fmt.Sprintf(
				"✅ *Бронь успешно подтверждена!*\n"+
					"━━━━━━━━━━━━━━━\n"+
					" Зал: `%s` | Стол: `%s`\n"+
					" Время: `%s`\n"+
					" Статус: *Подтверждено*\n\n"+
					"Ждем вас в гости!",
				booking.Zone, booking.Table, booking.TimeSlot,
			)

			_, err = b.Send(user, text, telegram.BuildMainMenu(), tele.ModeMarkdown)
			if err != nil {
				log.Printf("Ошибка при отправке сообщения: %v", err)
			}

			// 4. Уведомляем админа (теперь с именем и телефоном!)
			if adminID != 0 {
				notifyText := fmt.Sprintf(
					"🔔 *НОВАЯ БРОНЬ В СИСТЕМЕ*\n"+
						"━━━━━━━━━━━━━━━\n"+
						"👤 *Имя:* %s\n"+
						"📞 *Телефон:* %s\n"+
						"🆔 Гость ID: `%d`\n"+
						"📍 Зал: *%s* | Стол: *%s*\n"+
						"⏰ Время: *%s*",
					data.Name, data.Phone, data.UserID, booking.Zone, booking.Table, booking.TimeSlot,
				)
				_, _ = b.Send(&tele.User{ID: adminID}, notifyText, tele.ModeMarkdown)
			}

			w.WriteHeader(http.StatusOK)
		})

		// --- ЭНДПОИНТ ДЛЯ ПРОВЕРКИ ЗАНЯТОСТИ ---
		http.HandleFunc("/api/availability", func(w http.ResponseWriter, r *http.Request) {
			ctx := context.Background()
			bookings, err := bookingService.GetAllActiveBookings(ctx)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			bookedMap := make(map[string][]string)
			for _, b := range bookings {
				if b.TimeSlot != "" {
					bookedMap[b.Table] = append(bookedMap[b.Table], b.TimeSlot)
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bookedMap)
		})
		// ---------------------------------------------

		// 1. Отдаем статические файлы (картинки) по пути /img/
		http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("./webapp/img"))))

		// 2. Обрабатываем главную страницу через шаблонизатор
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}

			tmpl, err := template.ParseGlob("webapp/templates/*.html")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

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
