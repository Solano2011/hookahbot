package memory_test

import (
	"context"
	"errors"
	"testing"

	"hookah-bot/internal/domain"
	"hookah-bot/internal/repository/memory"
)

func TestBookingRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewBookingRepo()
	const userID = int64(101)

	// 1. Попытка получить несуществующую запись
	_, err := repo.GetByUserID(ctx, userID)
	if !errors.Is(err, domain.ErrBookingNotFound) {
		t.Fatalf("ожидалась ошибка ErrBookingNotFound, получено: %v", err)
	}

	// 2. Создание черновика и финализация
	if err := repo.SaveDraft(ctx, userID, "VIP-комната"); err != nil {
		t.Fatalf("ошибка сохранения драфта: %v", err)
	}

	saved, err := repo.CompleteBooking(ctx, userID, "22:00")
	if err != nil {
		t.Fatalf("ошибка подтверждения брони: %v", err)
	}
	if saved.Zone != "VIP-комната" || saved.TimeSlot != "22:00" {
		t.Errorf("данные не совпадают: %+v", saved)
	}

	// 3. Попытка занять этот же слот вторым пользователем
	const user2ID = int64(102)
	_ = repo.SaveDraft(ctx, user2ID, "VIP-комната")
	_, err = repo.CompleteBooking(ctx, user2ID, "22:00")
	if !errors.Is(err, domain.ErrTimeSlotTaken) {
		t.Errorf("ожидалась ошибка ErrTimeSlotTaken, получено: %v", err)
	}

	// 4. Удаление (отмена брони)
	if err := repo.Delete(ctx, userID); err != nil {
		t.Fatalf("ошибка удаления: %v", err)
	}

	_, err = repo.GetByUserID(ctx, userID)
	if !errors.Is(err, domain.ErrBookingNotFound) {
		t.Fatalf("после удаления ожидался ErrBookingNotFound, получено: %v", err)
	}
}
