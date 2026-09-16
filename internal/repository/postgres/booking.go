package postgres

import (
	"context"
	"errors"
	"time"

	"hookah-bot/internal/domain"

	"github.com/jackc/pgx/v5"
)

type BookingRepo struct {
	db *DB
}

// NewBookingRepo принимает наше подключение к БД
func NewBookingRepo(db *DB) *BookingRepo {
	return &BookingRepo{db: db}
}

func (r *BookingRepo) SaveDraft(ctx context.Context, userID int64, zone string) error {
	// ВАЖНО: Так как в БД стоит внешний ключ (REFERENCES users(id)),
	// нам нужно гарантировать, что пользователь существует в таблице users.
	// Добавляем его "тихо", если его еще нет:
	_, err := r.db.Conn.Exec(ctx, `INSERT INTO users (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, userID)
	if err != nil {
		return err
	}

	// Удаляем старый черновик, если он был (чтобы не плодить мусор)
	_, err = r.db.Conn.Exec(ctx, `DELETE FROM bookings WHERE user_id = $1 AND status = 'draft'`, userID)
	if err != nil {
		return err
	}

	// Создаем новый черновик. Пустые строки '' заменяют нам будущие стол и время
	_, err = r.db.Conn.Exec(ctx, `
        INSERT INTO bookings (user_id, zone, table_name, time_slot, status) 
        VALUES ($1, $2, '', '', 'draft')`,
		userID, zone,
	)
	return err
}

func (r *BookingRepo) SetTable(ctx context.Context, userID int64, table string) error {
	// Обновляем стол только у черновика
	cmdTag, err := r.db.Conn.Exec(ctx, `
        UPDATE bookings SET table_name = $1 
        WHERE user_id = $2 AND status = 'draft'`,
		table, userID,
	)
	if err != nil {
		return err
	}

	// Если ни одна строка не обновилась, значит черновика нет
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrBookingNotFound
	}
	return nil
}

func (r *BookingRepo) CompleteBooking(ctx context.Context, userID int64, timeSlot string, name string, phone string) (*domain.Booking, error) {
	// Используем транзакцию для предотвращения race condition
	tx, err := r.db.Conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 1. Получаем текущий черновик, чтобы узнать зону и стол
	var zone, table string
	err = tx.QueryRow(ctx, `
        SELECT zone, table_name FROM bookings
        WHERE user_id = $1 AND status = 'draft'`, userID).Scan(&zone, &table)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBookingNotFound
	} else if err != nil {
		return nil, err
	}

	// 2. Проверяем, не занял ли кто-то этот стол на это же время (FOR UPDATE блокирует конфликтующие строки)
	var conflictID int
	err = tx.QueryRow(ctx, `
        SELECT id FROM bookings
        WHERE zone = $1 AND table_name = $2 AND time_slot = $3 AND status = 'confirmed'
        FOR UPDATE`,
		zone, table, timeSlot).Scan(&conflictID)

	if err == nil {
		// Если err == nil, значит такая запись НАШЛАСЬ, стол занят!
		return nil, domain.ErrTimeSlotTaken
	} else if !errors.Is(err, pgx.ErrNoRows) {
		// Если ошибка какая-то другая (не "нет строк"), возвращаем её
		return nil, err
	}

	// 3. Стол свободен. Обновляем статус черновика на confirmed и сохраняем контакты
	var b domain.Booking
	err = tx.QueryRow(ctx, `
        UPDATE bookings
        SET time_slot = $1, user_name = $2, phone = $3, status = 'confirmed', created_at = $4
        WHERE user_id = $5 AND status = 'draft'
        RETURNING user_id, zone, table_name, time_slot, user_name, phone, created_at`,
		timeSlot, name, phone, time.Now(), userID,
	).Scan(&b.UserID, &b.Zone, &b.Table, &b.TimeSlot, &b.UserName, &b.Phone, &b.CreatedAt)

	if err != nil {
		return nil, err
	}

	// Коммитим транзакцию
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &b, nil
}

func (r *BookingRepo) GetByUserID(ctx context.Context, userID int64) (*domain.Booking, error) {
	var b domain.Booking
	// Ищем только подтвержденные брони. Берем самую последнюю (ORDER BY ... DESC)
	err := r.db.Conn.QueryRow(ctx, `
        SELECT user_id, zone, table_name, time_slot, COALESCE(user_name, ''), COALESCE(phone, ''), created_at
        FROM bookings
        WHERE user_id = $1 AND status = 'confirmed'
        ORDER BY created_at DESC LIMIT 1`,
		userID,
	).Scan(&b.UserID, &b.Zone, &b.Table, &b.TimeSlot, &b.UserName, &b.Phone, &b.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBookingNotFound
	} else if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BookingRepo) Delete(ctx context.Context, userID int64) error {
	// Удаляем все записи пользователя (и черновики, и готовые)
	_, err := r.db.Conn.Exec(ctx, `DELETE FROM bookings WHERE user_id = $1`, userID)
	return err
}

func (r *BookingRepo) GetAllActive(ctx context.Context) ([]domain.Booking, error) {
	rows, err := r.db.Conn.Query(ctx, `
        SELECT user_id, zone, table_name, time_slot, COALESCE(user_name, ''), COALESCE(phone, ''), created_at
        FROM bookings
        WHERE status = 'confirmed'
        ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Booking
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(&b.UserID, &b.Zone, &b.Table, &b.TimeSlot, &b.UserName, &b.Phone, &b.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, nil
}

func (r *BookingRepo) ResetAll(ctx context.Context) error {
	_, err := r.db.Conn.Exec(ctx, `TRUNCATE TABLE bookings`)
	return err
}
