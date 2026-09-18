package app

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	"hookah-bot/internal/delivery/telegram"
	"hookah-bot/internal/repository/postgres"
	"hookah-bot/internal/service"
	"hookah-bot/internal/validation"

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
	handlers := telegram.NewHandlers(bookingService, adminID, b)
	handlers.InitRoutes(b)

	log.Printf("Бот @%s успешно запущен! Admin ID: %d", b.Me.Username, adminID)

	// --- ЗАПУСК ВЕБ-СЕРВЕРА ---
	go func() {
		// Rate limiting map (потокобезопасная реализация)
		var rateLimitMap sync.Map

		http.HandleFunc("/api/book", func(w http.ResponseWriter, r *http.Request) {
			// CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			if r.Method != "POST" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var data struct {
				Table    string `json:"table"`
				Time     string `json:"time"`
				Date     string `json:"date"`
				UserID   int64  `json:"userId"`
				Name     string `json:"name"`
				Phone    string `json:"phone"`
				InitData string `json:"initData"` // Данные для проверки подписи
			}

			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				http.Error(w, "Invalid request format", http.StatusBadRequest)
				return
			}

			// КРИТИЧНО: Проверка подлинности запроса от Telegram WebApp
			if data.InitData != "" {
				if err := validation.VerifyTelegramWebAppData(data.InitData, token); err != nil {
					log.Printf("⚠️ Неверная подпись WebApp: %v", err)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}

			// Rate limiting: не более 1 запроса в 5 секунд от одного пользователя
			if val, ok := rateLimitMap.Load(data.UserID); ok {
				lastReq := val.(time.Time)
				if time.Since(lastReq) < 5*time.Second {
					http.Error(w, "Too many requests", http.StatusTooManyRequests)
					return
				}
			}
			rateLimitMap.Store(data.UserID, time.Now())

			// Валидация входных данных
			if err := validation.ValidateName(data.Name); err != nil {
				http.Error(w, fmt.Sprintf("Invalid name: %v", err), http.StatusBadRequest)
				return
			}

			if err := validation.ValidatePhone(data.Phone); err != nil {
				http.Error(w, fmt.Sprintf("Invalid phone: %v", err), http.StatusBadRequest)
				return
			}

			if err := validation.ValidateTableName(data.Table); err != nil {
				http.Error(w, fmt.Sprintf("Invalid table: %v", err), http.StatusBadRequest)
				return
			}

			if err := validation.ValidateTimeSlot(data.Time); err != nil {
				http.Error(w, fmt.Sprintf("Invalid time slot: %v", err), http.StatusBadRequest)
				return
			}

			if err := validation.ValidateDate(data.Date); err != nil {
				http.Error(w, fmt.Sprintf("Invalid date: %v", err), http.StatusBadRequest)
				return
			}

			ctx := context.Background()

			log.Printf("🌐 [HTTP API] Получен запрос на бронь от userID=%d: стол=%s, дата=%s, время=%s", data.UserID, data.Table, data.Date, data.Time)

			// --- НОВАЯ ЛОГИКА ЗАМЕНЫ БРОНИ ---
			existingBooking, err := bookingService.GetUserBooking(ctx, data.UserID)
			if err == nil && existingBooking.TimeSlot != "" {
				log.Printf("⚠️ [HTTP API] У userID=%d найдена существующая бронь: зал=%s, стол=%s, время=%s",
					data.UserID, existingBooking.Zone, existingBooking.Table, existingBooking.TimeSlot)

				// НЕ удаляем бронь, а сохраняем новые данные в черновик
				if err := bookingService.StartBookingDraft(ctx, data.UserID, "Общий лаунж"); err != nil {
					log.Printf("❌ [HTTP API] Ошибка создания черновика для userID=%d: %v", data.UserID, err)
					http.Error(w, "Failed to create booking draft", http.StatusInternalServerError)
					return
				}

				if err := bookingService.SetBookingTable(ctx, data.UserID, data.Table); err != nil {
					log.Printf("❌ [HTTP API] Ошибка сохранения стола в черновик для userID=%d: %v", data.UserID, err)
					http.Error(w, "Failed to save table to draft", http.StatusInternalServerError)
					return
				}

				if err := repo.SetDraftDate(ctx, data.UserID, data.Date); err != nil {
					log.Printf("❌ [HTTP API] Ошибка сохранения даты в черновик для userID=%d: %v", data.UserID, err)
					http.Error(w, "Failed to save date to draft", http.StatusInternalServerError)
					return
				}

				if err := bookingService.SetDraftTimeAndContacts(ctx, data.UserID, data.Time, data.Name, data.Phone); err != nil {
					log.Printf("❌ [HTTP API] Ошибка сохранения времени и контактов в черновик для userID=%d: %v", data.UserID, err)
					http.Error(w, "Failed to save time and contacts to draft", http.StatusInternalServerError)
					return
				}

				log.Printf("✅ [HTTP API] Черновик сохранён для userID=%d, ожидаем подтверждения", data.UserID)

				// Отправляем пользователю сообщение с выбором в Telegram
				user := &tele.User{ID: data.UserID}
				text := fmt.Sprintf(
					"⚠️ *У вас уже есть активная бронь:*\n\n"+
						"📍 Зал: `%s`\n"+
						"🪑 Стол: `%s`\n"+
						"📅 Дата: `%s`\n"+
						"⏰ Время: `%s`\n\n"+
						"Хотите отменить предыдущую бронь и создать новую на `%s` (`%s`) в `%s`?",
					existingBooking.Zone, existingBooking.Table, existingBooking.Date, existingBooking.TimeSlot,
					data.Table, data.Date, data.Time,
				)

				_, sendErr := b.Send(user, text, telegram.BuildReplaceConfirmMenu(), tele.ModeMarkdown)
				if sendErr != nil {
					log.Printf("⚠️ Ошибка при отправке сообщения выбора для userID=%d: %v", data.UserID, sendErr)
				}

				// Возвращаем статус фронтенду Web App, чтобы он понял, что нужно ждать подтверждения
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"status": "pending_confirmation"})
				return
			}

			log.Printf("ℹ️ [HTTP API] У userID=%d нет существующих броней, создаем новую", data.UserID)

			// --- СТАНДАРТНАЯ ЛОГИКА (ЕСЛИ СТАРОЙ БРОНИ НЕТ) ---

			// 0. СОЗДАЕМ ЧЕРНОВИК НА ЛЕТУ
			if err := bookingService.StartBookingDraft(ctx, data.UserID, "Общий лаунж"); err != nil {
				log.Printf("❌ [HTTP API] Ошибка создания черновика для userID=%d: %v", data.UserID, err)
				http.Error(w, "Failed to create booking draft", http.StatusInternalServerError)
				return
			}
			log.Printf("✅ [HTTP API] Черновик создан для userID=%d", data.UserID)

			// 1. СОХРАНЯЕМ СТОЛ В СЕРВИС
			if err := bookingService.SetBookingTable(ctx, data.UserID, data.Table); err != nil {
				log.Printf("❌ [HTTP API] Ошибка сохранения стола для userID=%d: %v", data.UserID, err)
				http.Error(w, "Failed to save table", http.StatusInternalServerError)
				return
			}
			log.Printf("✅ [HTTP API] Стол сохранён для userID=%d: %s", data.UserID, data.Table)

			// 1.5. СОХРАНЯЕМ ДАТУ В ЧЕРНОВИК
			if err := repo.SetDraftDate(ctx, data.UserID, data.Date); err != nil {
				log.Printf("❌ [HTTP API] Ошибка сохранения даты для userID=%d: %v", data.UserID, err)
				http.Error(w, "Failed to save date", http.StatusInternalServerError)
				return
			}
			log.Printf("✅ [HTTP API] Дата сохранена для userID=%d: %s", data.UserID, data.Date)

			// 2. ФИНАЛИЗИРУЕМ БРОНЬ (сохраняем время)
			booking, err := bookingService.CompleteBookingDraft(ctx, data.UserID, data.Time, data.Name, data.Phone)
			if err != nil {
				log.Printf("❌ [HTTP API] Ошибка завершения брони для userID=%d: %v", data.UserID, err)
				http.Error(w, "Booking conflict or error", http.StatusConflict)
				return
			}

			log.Printf("✅ [HTTP API] Бронь успешно создана для userID=%d: зал=%s, стол=%s, время=%s",
				data.UserID, booking.Zone, booking.Table, booking.TimeSlot)

			// 3. Отправляем красивое сообщение пользователю
			user := &tele.User{ID: data.UserID}
			text := fmt.Sprintf(
				"✅ *Бронь успешно подтверждена!*\n"+
					"━━━━━━━━━━━━━━━\n"+
					"📍 Зал: `%s` | Стол: `%s`\n"+
					"📅 Дата: `%s`\n"+
					"⏰ Время: `%s`\n"+
					"✨ Статус: *Подтверждено*\n\n"+
					"Ждем вас в гости!",
				booking.Zone, booking.Table, booking.Date, booking.TimeSlot,
			)

			_, err = b.Send(user, text, telegram.BuildMainMenu(), tele.ModeMarkdown)
			if err != nil {
				log.Printf("⚠️ Ошибка при отправке сообщения: %v", err)
			}

			// 4. Уведомляем админа
			if adminID != 0 {
				notifyText := fmt.Sprintf(
					"🔔 *НОВАЯ БРОНЬ В СИСТЕМЕ*\n"+
						"━━━━━━━━━━━━━━━\n"+
						"👤 *Имя:* %s\n"+
						"📞 *Телефон:* %s\n"+
						"🆔 Гость ID: `%d`\n"+
						"📍 Зал: *%s* | Стол: *%s*\n"+
						"📅 Дата: *%s*\n"+
						"⏰ Время: *%s*",
					data.Name, data.Phone, data.UserID, booking.Zone, booking.Table, booking.Date, booking.TimeSlot,
				)
				_, _ = b.Send(&tele.User{ID: adminID}, notifyText, tele.ModeMarkdown)
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		})

		// --- ЭНДПОИНТ ДЛЯ ПРОВЕРКИ ЗАНЯТОСТИ ---
		http.HandleFunc("/api/availability", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			if r.Method != "GET" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			date := r.URL.Query().Get("date")
			if date == "" {
				http.Error(w, "Missing date parameter", http.StatusBadRequest)
				return
			}

			ctx := context.Background()
			takenSlots, err := repo.GetTakenTimeSlots(ctx, date)
			if err != nil {
				log.Printf("❌ Ошибка получения занятых слотов для даты %s: %v", date, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(takenSlots)
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
				log.Printf("Ошибка загрузки шаблонов: %v", err)
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
