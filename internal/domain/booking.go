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
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	UserName  string    `json:"name"`  // Изменили тег на "name" под фронтенд
	Phone     string    `json:"phone"` // Тег "phone"
	Zone      string    `json:"zone"`
	Table     string    `json:"table"`
	TimeSlot  string    `json:"timeslot"` // Убедись, что здесь TimeSlot с большой S
	CreatedAt time.Time `json:"created_at"`
}

type BookingRepository interface {
	SaveDraft(ctx context.Context, userID int64, zone string) error
	SetTable(ctx context.Context, userID int64, table string) error
	CompleteBooking(ctx context.Context, userID int64, timeSlot string, name string, phone string) (*Booking, error) // <-- Добавили name и phone
	GetByUserID(ctx context.Context, userID int64) (*Booking, error)
	Delete(ctx context.Context, userID int64) error
	GetAllActive(ctx context.Context) ([]Booking, error)
	ResetAll(ctx context.Context) error
}

type BookingService interface {
	StartBookingDraft(ctx context.Context, userID int64, zone string) error
	SetBookingTable(ctx context.Context, userID int64, table string) error
	CompleteBookingDraft(ctx context.Context, userID int64, timeSlot string, name string, phone string) (*Booking, error)
	GetUserBooking(ctx context.Context, userID int64) (*Booking, error)
	CancelBooking(ctx context.Context, userID int64) error
	GetAllActiveBookings(ctx context.Context) ([]Booking, error)
	ResetAllBookings(ctx context.Context) error
}
