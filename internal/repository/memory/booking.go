package memory

import (
	"context"
	"sync"
	"time"

	"hookah-bot/internal/domain"
)

type BookingRepo struct {
	mu       sync.Mutex
	drafts   map[int64]domain.Booking // Временные черновики
	bookings map[int64]domain.Booking // Подтвержденные брони
}

func NewBookingRepo() *BookingRepo {
	return &BookingRepo{
		drafts:   make(map[int64]domain.Booking),
		bookings: make(map[int64]domain.Booking),
	}
}

func (r *BookingRepo) SaveDraft(ctx context.Context, userID int64, zone string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Сохраняем только в черновики! Готовая бронь (если есть) не страдает
	r.drafts[userID] = domain.Booking{
		UserID: userID,
		Zone:   zone,
	}
	return nil
}

func (r *BookingRepo) SetTable(ctx context.Context, userID int64, table string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	draft, ok := r.drafts[userID]
	if !ok {
		return domain.ErrBookingNotFound
	}

	draft.Table = table
	r.drafts[userID] = draft
	return nil
}

func (r *BookingRepo) CompleteBooking(ctx context.Context, userID int64, timeSlot string) (*domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	draft, ok := r.drafts[userID]
	if !ok {
		return nil, domain.ErrBookingNotFound
	}

	// Проверяем занятость среди ПОДТВЕРЖДЕННЫХ броней
	for _, b := range r.bookings {
		// Убрали проверку b.UserID != userID. Стол занят = стол занят для всех!
		if b.Zone == draft.Zone && b.Table == draft.Table && b.TimeSlot == timeSlot {
			return nil, domain.ErrTimeSlotTaken
		}
	}

	// Завершаем оформление
	draft.TimeSlot = timeSlot
	draft.CreatedAt = time.Now()

	// Переносим из черновиков в готовые брони
	r.bookings[userID] = draft
	delete(r.drafts, userID)

	res := draft
	return &res, nil
}

func (r *BookingRepo) GetByUserID(ctx context.Context, userID int64) (*domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ищем только среди готовых броней
	b, ok := r.bookings[userID]
	if !ok {
		return nil, domain.ErrBookingNotFound
	}

	res := b
	return &res, nil
}

func (r *BookingRepo) Delete(ctx context.Context, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.bookings, userID)
	delete(r.drafts, userID) // На всякий случай чистим и черновик
	return nil
}

func (r *BookingRepo) GetAllActive(ctx context.Context) ([]domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []domain.Booking
	for _, b := range r.bookings {
		result = append(result, b)
	}
	return result, nil
}

func (r *BookingRepo) ResetAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.bookings = make(map[int64]domain.Booking)
	r.drafts = make(map[int64]domain.Booking)
	return nil
}
