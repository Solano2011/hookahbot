package service_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"hookah-bot/internal/domain"
	"hookah-bot/internal/repository/memory"
	"hookah-bot/internal/service"
)

func TestBookingSvc_ConcurrentBooking(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewBookingRepo()
	svc := service.NewBookingService(repo)

	const (
		goroutines = 50
		targetZone = "VIP-комната"
		targetTime = "22:00"
	)

	// Создаем черновики для 50 разных пользователей
	for i := 1; i <= goroutines; i++ {
		err := svc.StartBookingDraft(ctx, int64(i), targetZone)
		if err != nil {
			t.Fatalf("не удалось создать драфт для user %d: %v", i, err)
		}
	}

	var (
		wg           sync.WaitGroup
		successCount int64
		takenCount   int64
		startBarrier = make(chan struct{}) // Барьер для одновременного старта горутин
	)

	wg.Add(goroutines)
	for i := 1; i <= goroutines; i++ {
		userID := int64(i)
		go func() {
			defer wg.Done()

			// Ждем отмашки, чтобы все 50 горутин ударили одновременно
			<-startBarrier

			_, err := svc.CompleteBookingDraft(ctx, userID, targetTime)
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			} else if errors.Is(err, domain.ErrTimeSlotTaken) {
				atomic.AddInt64(&takenCount, 1)
			}
		}()
	}

	// Снимаем барьер
	close(startBarrier)
	wg.Wait()

	t.Logf("Успешных броней: %d, Отклоненных (занято): %d", successCount, takenCount)

	if successCount != 1 {
		t.Errorf("Овербукинг! Ожидалась ровно 1 успешная бронь, получено: %d", successCount)
	}

	if takenCount != int64(goroutines-1) {
		t.Errorf("Ожидалось %d отказов, получено: %d", goroutines-1, takenCount)
	}
}
