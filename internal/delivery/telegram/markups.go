package telegram

import tele "gopkg.in/telebot.v3"

var (
	Menu = &tele.ReplyMarkup{}

	// Главное меню
	BtnBook      = Menu.Data(" Забронировать стол", "btn_book")
	BtnMenu      = Menu.Data(" Меню & Табачная карта", "btn_menu")
	BtnMyBooking = Menu.Data(" Моя бронь", "btn_my_booking")
	BtnContacts  = Menu.Data(" Локация & Контакты", "btn_contacts")

	// Навигация
	BtnBackToMain    = Menu.Data(" Назад в меню", "btn_back_main")
	BtnCancelBooking = Menu.Data(" Отменить бронь", "btn_cancel_booking")

	// Эндпоинты
	BtnZone  = Menu.Data("", "zone")
	BtnTable = Menu.Data("", "table") // Наша новая кнопка для столов
	BtnTime  = Menu.Data("", "time")

	// Админка
	BtnAdminRefresh  = Menu.Data(" Обновить сводку", "admin_refresh")
	BtnAdminResetAll = Menu.Data(" Сбросить все слоты", "admin_reset_all")
)

func BuildMainMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(BtnBook),
		m.Row(BtnMenu, BtnMyBooking),
		m.Row(BtnContacts),
	)
	return m
}

func BuildZonesMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(m.Data(" Общий лаунж • Атмосферный зал", "zone", "Общий лаунж")),
		m.Row(m.Data(" PS5 Lounge • 4K TV & Звук", "zone", "Зона с PS5")),
		m.Row(m.Data(" VIP-комната • До 8 человек", "zone", "VIP-комната")),
		m.Row(BtnBackToMain),
	)
	return m
}

func BuildTablesMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(m.Data("Стол 1", "table", "Стол 1"), m.Data("Стол 2", "table", "Стол 2")),
		m.Row(m.Data("Стол 3", "table", "Стол 3"), m.Data("Стол 4", "table", "Стол 4")),
		m.Row(m.Data("👑 У окна (Стол 5)", "table", "Стол 5")),
		m.Row(BtnBackToMain),
	)
	return m
}

func BuildTimeMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(m.Data(" 18:00", "time", "18:00"), m.Data(" 20:00", "time", "20:00")),
		m.Row(m.Data(" 22:00", "time", "22:00"), m.Data(" 00:00", "time", "00:00")),
		m.Row(BtnBackToMain),
	)
	return m
}

func BuildContactsMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	btnMap := m.URL(" Открыть на Яндекс.Картах", "https://yandex.ru/maps")
	m.Inline(
		m.Row(btnMap),
		m.Row(BtnBackToMain),
	)
	return m
}

func BuildAdminMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(BtnAdminRefresh),
		m.Row(BtnAdminResetAll),
	)
	return m
}
