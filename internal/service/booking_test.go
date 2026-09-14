package service_test

import (
	"context"
	"errors"
	"testing"

	"hookah-bot/internal/domain"
	"hookah-bot/internal/service"
)

type mockBookingRepo struct {
	saveDraftFunc       func(ctx context.Context, userID int64, zone string) error
	setTableFunc        func(ctx context.Context, userID int64, table string) error // Добавлен мок-метод
	completeBookingFunc func(ctx context.Context, userID int64, timeSlot string) (*domain.Booking, error)
	getByUserIDFunc     func(ctx context.Context, userID int64) (*domain.Booking, error)
	deleteFunc          func(ctx context.Context, userID int64) error
	getAllActiveFunc    func(ctx context.Context) ([]domain.Booking, error)
	resetAllFunc        func(ctx context.Context) error
}

func (m *mockBookingRepo) SaveDraft(ctx context.Context, userID int64, zone string) error {
	if m.saveDraftFunc != nil {
		return m.saveDraftFunc(ctx, userID, zone)
	}
	return nil
}

// Реализация нового метода из интерфейса
func (m *mockBookingRepo) SetTable(ctx context.Context, userID int64, table string) error {
	if m.setTableFunc != nil {
		return m.setTableFunc(ctx, userID, table)
	}
	return nil
}

func (m *mockBookingRepo) CompleteBooking(ctx context.Context, userID int64, timeSlot string) (*domain.Booking, error) {
	if m.completeBookingFunc != nil {
		return m.completeBookingFunc(ctx, userID, timeSlot)
	}
	return nil, nil
}

func (m *mockBookingRepo) GetByUserID(ctx context.Context, userID int64) (*domain.Booking, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockBookingRepo) Delete(ctx context.Context, userID int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, userID)
	}
	return nil
}

func (m *mockBookingRepo) GetAllActive(ctx context.Context) ([]domain.Booking, error) {
	if m.getAllActiveFunc != nil {
		return m.getAllActiveFunc(ctx)
	}
	return nil, nil
}

func (m *mockBookingRepo) ResetAll(ctx context.Context) error {
	if m.resetAllFunc != nil {
		return m.resetAllFunc(ctx)
	}
	return nil
}

func TestBookingSvc_StartBookingDraft(t *testing.T) {
	ctx := context.Background()
	const expectedUserID = int64(12345)
	const expectedZone = "VIP-комната"

	var savedUser int64
	var savedZone string

	mockRepo := &mockBookingRepo{
		saveDraftFunc: func(ctx context.Context, userID int64, zone string) error {
			savedUser = userID
			savedZone = zone
			return nil
		},
	}

	svc := service.NewBookingService(mockRepo)
	err := svc.StartBookingDraft(ctx, expectedUserID, expectedZone)

	if err != nil {
		t.Fatalf("ожидалось nil, получена ошибка: %v", err)
	}
	if savedUser != expectedUserID || savedZone != expectedZone {
		t.Errorf("некорректные данные: user=%d, zone=%s", savedUser, savedZone)
	}
}

func TestBookingSvc_CompleteBookingDraft(t *testing.T) {
	ctx := context.Background()
	const userID = int64(999)

	tests := []struct {
		name         string
		completeRes  *domain.Booking
		completeErr  error
		expectedErr  error
		expectedTime string
	}{
		{
			name:         "Успешное завершение брони",
			completeRes:  &domain.Booking{UserID: userID, Zone: "PS5", TimeSlot: "20:00"},
			completeErr:  nil,
			expectedErr:  nil,
			expectedTime: "20:00",
		},
		{
			name:        "Черновик не найден",
			completeRes: nil,
			completeErr: domain.ErrBookingNotFound,
			expectedErr: domain.ErrBookingNotFound,
		},
		{
			name:        "Слот времени уже занят",
			completeRes: nil,
			completeErr: domain.ErrTimeSlotTaken,
			expectedErr: domain.ErrTimeSlotTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockBookingRepo{
				completeBookingFunc: func(ctx context.Context, uID int64, slot string) (*domain.Booking, error) {
					return tt.completeRes, tt.completeErr
				},
			}

			svc := service.NewBookingService(repo)
			res, err := svc.CompleteBookingDraft(ctx, userID, "20:00")

			if !errors.Is(err, tt.expectedErr) {
				t.Fatalf("ожидалась ошибка %v, получена %v", tt.expectedErr, err)
			}

			if tt.expectedErr == nil {
				if res == nil {
					t.Fatal("ожидалась бронь, получен nil")
				}
				if res.TimeSlot != tt.expectedTime {
					t.Errorf("ожидался слот %s, получен %s", tt.expectedTime, res.TimeSlot)
				}
			}
		})
	}
}

func TestBookingSvc_CancelBooking(t *testing.T) {
	ctx := context.Background()
	const userID = int64(777)
	deleted := false

	repo := &mockBookingRepo{
		deleteFunc: func(ctx context.Context, id int64) error {
			if id == userID {
				deleted = true
			}
			return nil
		},
	}

	svc := service.NewBookingService(repo)
	err := svc.CancelBooking(ctx, userID)

	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if !deleted {
		t.Error("ожидалось удаление записи из репозитория")
	}
}

func TestBookingSvc_AdminMethods(t *testing.T) {
	ctx := context.Background()
	activeCalled := false
	resetCalled := false

	repo := &mockBookingRepo{
		getAllActiveFunc: func(ctx context.Context) ([]domain.Booking, error) {
			activeCalled = true
			return []domain.Booking{{UserID: 1, Zone: "VIP", TimeSlot: "20:00"}}, nil
		},
		resetAllFunc: func(ctx context.Context) error {
			resetCalled = true
			return nil
		},
	}

	svc := service.NewBookingService(repo)

	bookings, err := svc.GetAllActiveBookings(ctx)
	if err != nil || len(bookings) != 1 || !activeCalled {
		t.Errorf("ошибка вызова GetAllActiveBookings")
	}

	if err := svc.ResetAllBookings(ctx); err != nil || !resetCalled {
		t.Errorf("ошибка вызова ResetAllBookings")
	}
}
