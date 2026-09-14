package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrBookingNotFound = errors.New("бронь не найдена")
	ErrTimeSlotTaken   = errors.New("это время уже занято")
)

type Booking struct {
	ID        string
	UserID    int64
	Zone      string
	Table     string // Новое поле для номера столика
	TimeSlot  string
	CreatedAt time.Time
}

type BookingRepository interface {
	SaveDraft(ctx context.Context, userID int64, zone string) error
	SetTable(ctx context.Context, userID int64, table string) error // Выбор стола
	CompleteBooking(ctx context.Context, userID int64, timeSlot string) (*Booking, error)
	GetByUserID(ctx context.Context, userID int64) (*Booking, error)
	Delete(ctx context.Context, userID int64) error
	GetAllActive(ctx context.Context) ([]Booking, error)
	ResetAll(ctx context.Context) error
}

type BookingService interface {
	StartBookingDraft(ctx context.Context, userID int64, zone string) error
	SetBookingTable(ctx context.Context, userID int64, table string) error
	CompleteBookingDraft(ctx context.Context, userID int64, timeSlot string) (*Booking, error)
	GetUserBooking(ctx context.Context, userID int64) (*Booking, error)
	CancelBooking(ctx context.Context, userID int64) error
	GetAllActiveBookings(ctx context.Context) ([]Booking, error)
	ResetAllBookings(ctx context.Context) error
}
