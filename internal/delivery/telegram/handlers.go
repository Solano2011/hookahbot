package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"hookah-bot/internal/domain"

	tele "gopkg.in/telebot.v3"
)

const (
	ImgHeroUrl = "https://images.unsplash.com/photo-1517248135467-4c7edcad34c4?w=900&auto=format&fit=crop&q=80"
)

type Handlers struct {
	bookingService domain.BookingService
	adminID        int64
	bot            *tele.Bot
}

func NewHandlers(bs domain.BookingService, adminID int64) *Handlers {
	return &Handlers{
		bookingService: bs,
		adminID:        adminID,
	}
}

func (h *Handlers) InitRoutes(b *tele.Bot) {
	h.bot = b

	b.Handle("/start", h.handleStart)
	b.Handle("/admin", h.handleAdmin)

	b.Handle(&BtnBackToMain, h.handleBackToMain)
	b.Handle(&BtnBook, h.handleBookBtn)
	b.Handle(&BtnMyBooking, h.handleMyBookingBtn)
	b.Handle(&BtnCancelBooking, h.handleCancelBooking)
	b.Handle(&BtnContacts, h.handleContactsBtn)

	b.Handle(&BtnZone, h.handleZoneSelect)
	b.Handle(&BtnTime, h.handleTimeSelect)

	// Обработчик данных из Web App
	b.Handle(tele.OnWebApp, h.handleWebApp)

	b.Handle(&BtnAdminRefresh, h.handleAdminRefresh)
	b.Handle(&BtnAdminResetAll, h.handleAdminResetAll)
}

func (h *Handlers) isAdmin(userID int64) bool {
	return h.adminID != 0 && userID == h.adminID
}

func (h *Handlers) handleStart(c tele.Context) error {
	caption := fmt.Sprintf(
		"Добро пожаловать в *SMOKE LOUNGE*, %s!\n\n"+
			"Камерная атмосфера, авторские миксы и приватные зоны для идеального отдыха.\n\n"+
			"Выберите действие в меню ниже:",
		c.Sender().FirstName,
	)

	photo := &tele.Photo{File: tele.FromURL(ImgHeroUrl), Caption: caption}
	return c.Send(photo, BuildMainMenu(), tele.ModeMarkdown)
}

func (h *Handlers) handleBackToMain(c tele.Context) error {
	_ = c.Delete()
	photo := &tele.Photo{File: tele.FromURL(ImgHeroUrl), Caption: "Главное меню *SMOKE LOUNGE*:"}
	return c.Send(photo, BuildMainMenu(), tele.ModeMarkdown)
}

func (h *Handlers) handleBookBtn(c tele.Context) error {
	_ = c.Delete()
	text := " *Шаг 1 из 2: Выберите зал*\n\n" +
		"• *Общий лаунж* — мягкие диваны, приглушенный свет, chillout-музыка.\n" +
		"• *PS5 Lounge* — зона с PlayStation 5, топовыми играми и 4K ТВ.\n" +
		"• *VIP-комната* — полная приватность, отдельная аудиосистема (до 8 гостей)."
	return c.Send(text, BuildZonesMenu(), tele.ModeMarkdown)
}

func (h *Handlers) handleZoneSelect(c tele.Context) error {
	zone := c.Data()
	ctx := context.Background()

	if err := h.bookingService.StartBookingDraft(ctx, c.Sender().ID, zone); err != nil {
		return c.Send("Ошибка сохранения. Попробуйте еще раз.")
	}

	_ = c.Delete()

	// Вызов Mini App для Общего лаунжа
	// Вызов Mini App для Общего лаунжа
	if zone == "Общий лаунж" {
		m := &tele.ReplyMarkup{}
		baseURL := "https://satirical-starlight-scraggly.ngrok-free.dev"

		btnBook := m.WebApp("Забронировать стол", &tele.WebApp{URL: baseURL})
		btnMenu := m.WebApp("Меню & Табачная карта", &tele.WebApp{URL: baseURL + "/?start=menu"})

		// Удалили локальные btnMyBook и btnLocation!
		// Вместо них используем глобальные BtnMyBooking и BtnContacts:
		m.Inline(
			m.Row(btnBook),
			m.Row(btnMenu, BtnMyBooking),
			m.Row(BtnContacts),
		)

		return c.Send("Главное меню SMOKE LOUNGE:", m)
	}

	// Для VIP и PS5 стол один
	_ = h.bookingService.SetBookingTable(ctx, c.Sender().ID, "Основной")
	text := fmt.Sprintf(" *Шаг 2 из 2: Выберите время*\n\nВыбранный зал: `%s`", zone)
	return c.Send(text, BuildTimeMenu(), tele.ModeMarkdown)
}

// Принимаем стол из Web App
func (h *Handlers) handleWebApp(c tele.Context) error {
	if c.Message().WebAppData == nil {
		return nil
	}

	rawData := c.Message().WebAppData.Data

	// Ожидаем строку вида "Стол 3|20:00"
	parts := strings.Split(rawData, "|")
	if len(parts) != 2 {
		return c.Send("Ошибка: Неверный формат данных от Web App.")
	}

	table := parts[0]
	timeSlot := parts[1]

	ctx := context.Background()
	userID := c.Sender().ID

	// 1. Сохраняем стол
	if err := h.bookingService.SetBookingTable(ctx, userID, table); err != nil {
		return c.Send("Ошибка сохранения стола.")
	}

	// 2. СРАЗУ сохраняем время (финализируем бронь)
	booking, err := h.bookingService.CompleteBookingDraft(ctx, userID, timeSlot)
	if err != nil {
		if errors.Is(err, domain.ErrTimeSlotTaken) {
			return c.Send("Этот слот уже занят! Начните бронирование заново.")
		}
		return c.Send("Сессия истекла или произошла ошибка. Начните заново.")
	}

	// Удаляем сообщение с кнопкой Web App
	_ = h.bot.Delete(c.Message())

	// Уведомляем админа
	if h.adminID != 0 && h.bot != nil {
		user := c.Sender()
		usernameStr := "@" + user.Username
		if user.Username == "" {
			usernameStr = "без username"
		}
		notifyText := fmt.Sprintf(
			" *НОВАЯ БРОНЬ В СИСТЕМЕ*\n"+
				"━━━━━━━━━━━━━━━\n"+
				" Гость: *%s* (%s)\n"+
				" Зал: *%s* | *%s*\n"+
				" Время: *%s*",
			user.FirstName, usernameStr, booking.Zone, booking.Table, booking.TimeSlot,
		)
		go func(msg string) { _, _ = h.bot.Send(tele.ChatID(h.adminID), msg, tele.ModeMarkdown) }(notifyText)
	}

	// Подтверждаем пользователю
	text := fmt.Sprintf(
		" *Бронь успешно подтверждена!*\n"+
			"━━━━━━━━━━━━━━━\n"+
			" Зал: `%s` | Стол: `%s`\n"+
			" Время: `%s`\n"+
			" Статус: *Подтверждено*\n\n"+
			"Ждем вас в гости!",
		booking.Zone, booking.Table, booking.TimeSlot,
	)

	return c.Send(text, BuildMainMenu(), tele.ModeMarkdown)
}

func (h *Handlers) handleTimeSelect(c tele.Context) error {
	timeSlot := c.Data()
	ctx := context.Background()

	booking, err := h.bookingService.CompleteBookingDraft(ctx, c.Sender().ID, timeSlot)
	if err != nil {
		if errors.Is(err, domain.ErrTimeSlotTaken) {
			return c.Respond(&tele.CallbackResponse{Text: " Этот слот уже занят! Выберите другое время.", ShowAlert: true})
		}
		return c.Send("Сессия истекла. Начните выбор заново.")
	}

	if h.adminID != 0 && h.bot != nil {
		user := c.Sender()
		usernameStr := "@" + user.Username
		if user.Username == "" {
			usernameStr = "без username"
		}
		notifyText := fmt.Sprintf(
			" *НОВАЯ БРОНЬ В СИСТЕМЕ*\n"+
				"━━━━━━━━━━━━━━━\n"+
				" Гость: *%s* (%s)\n"+
				" Зал: *%s* | *%s*\n"+
				" Время: *%s*",
			user.FirstName, usernameStr, booking.Zone, booking.Table, booking.TimeSlot,
		)
		go func(msg string) { _, _ = h.bot.Send(tele.ChatID(h.adminID), msg, tele.ModeMarkdown) }(notifyText)
	}

	_ = c.Delete()
	text := fmt.Sprintf(
		" *Бронь успешно подтверждена!*\n"+
			"━━━━━━━━━━━━━━━\n"+
			" Зал: `%s` | Стол: `%s`\n"+
			" Время: `%s`\n"+
			" Статус: *Подтверждено*\n\n"+
			"Ждем вас в гости!",
		booking.Zone, booking.Table, booking.TimeSlot,
	)

	return c.Send(text, BuildMainMenu(), tele.ModeMarkdown)
}

func (h *Handlers) handleMyBookingBtn(c tele.Context) error {
	ctx := context.Background()
	b, err := h.bookingService.GetUserBooking(ctx, c.Sender().ID)

	_ = c.Delete()
	if err != nil || b.TimeSlot == "" {
		m := &tele.ReplyMarkup{}
		m.Inline(m.Row(BtnBackToMain))
		return c.Send("У вас пока нет активных бронирований.", m)
	}

	m := &tele.ReplyMarkup{}
	m.Inline(m.Row(BtnCancelBooking), m.Row(BtnBackToMain))

	text := fmt.Sprintf(
		" *Ваша бронь:*\n━━━━━━━━━━━━━━━\n Зал: `%s` | Стол: `%s`\n Время: `%s`\n",
		b.Zone, b.Table, b.TimeSlot,
	)
	return c.Send(text, m, tele.ModeMarkdown)
}

func (h *Handlers) handleCancelBooking(c tele.Context) error {
	ctx := context.Background()
	_ = h.bookingService.CancelBooking(ctx, c.Sender().ID)
	_ = c.Delete()
	return c.Send(" Бронь успешно аннулирована.", BuildMainMenu())
}

func (h *Handlers) handleContactsBtn(c tele.Context) error {
	_ = c.Delete()
	text := " *Smoke Lounge Central*\n\n *Адрес:* ул. Центральная, д. 15\n *Телефон:* `+7 (999) 000-00-00`"
	return c.Send(text, BuildContactsMenu(), tele.ModeMarkdown)
}

func (h *Handlers) renderAdminDashboard(ctx context.Context) (string, error) {
	bookings, err := h.bookingService.GetAllActiveBookings(ctx)
	if err != nil {
		return "", err
	}
	if len(bookings) == 0 {
		return " *Панель администратора*\n\n На сегодня активных броней нет.", nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(" *Панель администратора*\nВсего активных броней: *%d*\n━━━━━━━━━━━━━━━\n", len(bookings)))
	for i, b := range bookings {
		sb.WriteString(fmt.Sprintf("*%d.* `%s` | *%s* - *%s* (Гость: `%d`)\n", i+1, b.TimeSlot, b.Zone, b.Table, b.UserID))
	}
	return sb.String(), nil
}

func (h *Handlers) handleAdmin(c tele.Context) error {
	if !h.isAdmin(c.Sender().ID) {
		return c.Send(" У вас нет прав администратора.")
	}
	text, _ := h.renderAdminDashboard(context.Background())
	return c.Send(text, BuildAdminMenu(), tele.ModeMarkdown)
}

func (h *Handlers) handleAdminRefresh(c tele.Context) error {
	if !h.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "Доступ запрещен", ShowAlert: true})
	}
	text, _ := h.renderAdminDashboard(context.Background())
	_ = c.Delete()
	return c.Send(text, BuildAdminMenu(), tele.ModeMarkdown)
}

func (h *Handlers) handleAdminResetAll(c tele.Context) error {
	if !h.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "Доступ запрещен", ShowAlert: true})
	}
	ctx := context.Background()
	_ = h.bookingService.ResetAllBookings(ctx)
	_ = c.Delete()
	return c.Send(" *Все брони успешно аннулированы.*", BuildAdminMenu(), tele.ModeMarkdown)
}
