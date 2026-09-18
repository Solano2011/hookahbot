package validation

import (
	"fmt"
	"time"
)

// ValidateDate проверяет, что дата в формате YYYY-MM-DD и находится в диапазоне 0-6 дней от сегодня
func ValidateDate(date string) error {
	if date == "" {
		return fmt.Errorf("date is empty")
	}

	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	today := time.Now().Truncate(24 * time.Hour)
	maxDate := today.AddDate(0, 0, 6) // 7 дней вперед (0-6)

	if parsedDate.Before(today) {
		return fmt.Errorf("date cannot be in the past")
	}

	if parsedDate.After(maxDate) {
		return fmt.Errorf("date cannot be more than 7 days in the future")
	}

	return nil
}
