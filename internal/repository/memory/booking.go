package memory

import (
	"context"
	"sync"
	"time"

	"hookah-bot/internal/domain"
)

type BookingRepo struct {
	mu       sync.Mutex
	bookings map[int64]domain.Booking
}

func NewBookingRepo() *BookingRepo {
	return &BookingRepo{
		bookings: make(map[int64]domain.Booking),
	}
}

func (r *BookingRepo) SaveDraft(ctx context.Context, userID int64, zone string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.bookings[userID] = domain.Booking{
		UserID: userID,
		Zone:   zone,
	}
	return nil
}

func (r *BookingRepo) SetTable(ctx context.Context, userID int64, table string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	draft, ok := r.bookings[userID]
	if !ok {
		return domain.ErrBookingNotFound
	}

	draft.Table = table
	r.bookings[userID] = draft
	return nil
}

func (r *BookingRepo) CompleteBooking(ctx context.Context, userID int64, timeSlot string) (*domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	draft, ok := r.bookings[userID]
	if !ok {
		return nil, domain.ErrBookingNotFound
	}

	// Проверяем занятость конкретного стола в конкретной зоне
	for _, b := range r.bookings {
		if b.UserID != userID && b.Zone == draft.Zone && b.Table == draft.Table && b.TimeSlot == timeSlot {
			return nil, domain.ErrTimeSlotTaken
		}
	}

	draft.TimeSlot = timeSlot
	draft.CreatedAt = time.Now()
	r.bookings[userID] = draft

	res := draft
	return &res, nil
}

func (r *BookingRepo) GetByUserID(ctx context.Context, userID int64) (*domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

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
	return nil
}

func (r *BookingRepo) GetAllActive(ctx context.Context) ([]domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []domain.Booking
	for _, b := range r.bookings {
		if b.TimeSlot != "" {
			result = append(result, b)
		}
	}
	return result, nil
}

func (r *BookingRepo) ResetAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bookings = make(map[int64]domain.Booking)
	return nil
}
