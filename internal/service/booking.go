package service

import (
	"context"
	"hookah-bot/internal/domain"
)

type BookingSvc struct {
	repo domain.BookingRepository
}

func NewBookingService(repo domain.BookingRepository) *BookingSvc {
	return &BookingSvc{repo: repo}
}

func (s *BookingSvc) StartBookingDraft(ctx context.Context, userID int64, zone string) error {
	return s.repo.SaveDraft(ctx, userID, zone)
}

func (s *BookingSvc) SetBookingTable(ctx context.Context, userID int64, table string) error {
	return s.repo.SetTable(ctx, userID, table)
}

func (s *BookingSvc) CompleteBookingDraft(ctx context.Context, userID int64, timeSlot string) (*domain.Booking, error) {
	return s.repo.CompleteBooking(ctx, userID, timeSlot)
}

func (s *BookingSvc) GetUserBooking(ctx context.Context, userID int64) (*domain.Booking, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *BookingSvc) CancelBooking(ctx context.Context, userID int64) error {
	return s.repo.Delete(ctx, userID)
}

func (s *BookingSvc) GetAllActiveBookings(ctx context.Context) ([]domain.Booking, error) {
	return s.repo.GetAllActive(ctx)
}

func (s *BookingSvc) ResetAllBookings(ctx context.Context) error {
	return s.repo.ResetAll(ctx)
}
