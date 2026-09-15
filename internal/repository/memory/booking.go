package memory

import (
	"context"
	"sync"

	"hookah-bot/internal/domain"
)

type BookingRepo struct {
	mu     sync.Mutex
	drafts map[int64]*domain.Booking
}

func NewBookingRepo() *BookingRepo {
	return &BookingRepo{
		drafts: make(map[int64]*domain.Booking),
	}
}

func (r *BookingRepo) SaveDraft(ctx context.Context, userID int64, zone string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.drafts[userID] = &domain.Booking{
		UserID: userID,
		Zone:   zone,
	}
	return nil
}

func (r *BookingRepo) SetTable(ctx context.Context, userID int64, table string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, exists := r.drafts[userID]
	if !exists {
		return domain.ErrBookingNotFound
	}
	b.Table = table
	return nil
}

func (r *BookingRepo) CompleteBooking(ctx context.Context, userID int64, timeSlot string, name string, phone string) (*domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, exists := r.drafts[userID]
	if !exists {
		return nil, domain.ErrBookingNotFound
	}

	for _, existing := range r.drafts {
		if existing.TimeSlot == timeSlot && existing.Table == b.Table && existing.UserID != userID {
			return nil, domain.ErrTimeSlotTaken
		}
	}

	b.TimeSlot = timeSlot
	b.UserName = name
	b.Phone = phone

	return b, nil // Возвращаем указатель
}

func (r *BookingRepo) GetByUserID(ctx context.Context, userID int64) (*domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, exists := r.drafts[userID]
	if !exists || b.TimeSlot == "" {
		return nil, domain.ErrBookingNotFound
	}
	return b, nil
}

func (r *BookingRepo) Delete(ctx context.Context, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.drafts, userID)
	return nil
}

func (r *BookingRepo) GetAllActive(ctx context.Context) ([]domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var active []domain.Booking
	for _, b := range r.drafts {
		if b.TimeSlot != "" {
			active = append(active, *b)
		}
	}
	return active, nil
}

func (r *BookingRepo) ResetAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.drafts = make(map[int64]*domain.Booking)
	return nil
}
