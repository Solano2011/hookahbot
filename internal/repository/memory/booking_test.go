package memory_test

import (
	"context"
	"testing"

	"hookah-bot/internal/repository/memory"
)

func TestMemoryBookingRepo_CompleteBooking(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewBookingRepo()

	userID := int64(123)
	zone := "Общий лаунж"
	table := "Стол 1"
	timeSlot := "20:00"

	// 1. Создаем черновик
	if err := repo.SaveDraft(ctx, userID, zone); err != nil {
		t.Fatalf("ошибка сохранения драфта: %v", err)
	}

	// 2. Устанавливаем стол
	if err := repo.SetTable(ctx, userID, table); err != nil {
		t.Fatalf("ошибка установки стола: %v", err)
	}

	// 3. Завершаем бронь с 5 аргументами (время, имя, телефон)
	booking, err := repo.CompleteBooking(ctx, userID, timeSlot, "Иван", "+79991112233")
	if err != nil {
		t.Fatalf("ошибка завершения брони: %v", err)
	}

	if booking.TimeSlot != timeSlot || booking.UserName != "Иван" || booking.Phone != "+79991112233" {
		t.Errorf("данные брони не совпали: %+v", booking)
	}
}
