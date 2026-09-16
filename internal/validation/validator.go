package validation

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var (
	phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	nameRegex  = regexp.MustCompile(`^[\p{L}\p{M}\s'-]+$`)
)

// ValidateName проверяет корректность имени
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("имя не может быть пустым")
	}

	if utf8.RuneCountInString(name) > 100 {
		return fmt.Errorf("имя слишком длинное (максимум 100 символов)")
	}

	if utf8.RuneCountInString(name) < 2 {
		return fmt.Errorf("имя слишком короткое (минимум 2 символа)")
	}

	if !nameRegex.MatchString(name) {
		return fmt.Errorf("имя содержит недопустимые символы")
	}

	return nil
}

// ValidatePhone проверяет корректность телефона
func ValidatePhone(phone string) error {
	if phone == "" {
		return fmt.Errorf("телефон не может быть пустым")
	}

	// Удаляем пробелы и дефисы для проверки
	cleaned := strings.ReplaceAll(phone, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")

	if !phoneRegex.MatchString(cleaned) {
		return fmt.Errorf("некорректный формат телефона")
	}

	return nil
}

// ValidateTimeSlot проверяет корректность временного слота
func ValidateTimeSlot(timeSlot string) error {
	validSlots := []string{"18:00", "19:00", "20:00", "21:00", "22:00", "23:00", "00:00"}

	for _, valid := range validSlots {
		if timeSlot == valid {
			return nil
		}
	}

	return fmt.Errorf("некорректный временной слот")
}

// ValidateTableName проверяет корректность названия стола
func ValidateTableName(table string) error {
	validTables := []string{
		"Стол 1", "Стол 2", "Стол 3",
		"Стол 4 (окно)", "Стол 5 (окно)", "Стол 6 (окно)",
		"Основной",
	}

	for _, valid := range validTables {
		if table == valid {
			return nil
		}
	}

	return fmt.Errorf("некорректное название стола")
}

// ValidateZone проверяет корректность зоны
func ValidateZone(zone string) error {
	validZones := []string{"Общий лаунж", "Зона с PS5", "VIP-комната"}

	for _, valid := range validZones {
		if zone == valid {
			return nil
		}
	}

	return fmt.Errorf("некорректная зона")
}

// VerifyTelegramWebAppData проверяет подпись данных от Telegram WebApp
// initData - строка initData из Telegram.WebApp.initData
// botToken - токен бота
func VerifyTelegramWebAppData(initData, botToken string) error {
	if initData == "" {
		return fmt.Errorf("отсутствуют данные инициализации")
	}

	values, err := url.ParseQuery(initData)
	if err != nil {
		return fmt.Errorf("некорректный формат данных")
	}

	hash := values.Get("hash")
	if hash == "" {
		return fmt.Errorf("отсутствует подпись")
	}
	values.Del("hash")

	// Создаем строку data-check-string
	var keys []string
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var dataCheckParts []string
	for _, key := range keys {
		dataCheckParts = append(dataCheckParts, fmt.Sprintf("%s=%s", key, values.Get(key)))
	}
	dataCheckString := strings.Join(dataCheckParts, "\n")

	// Вычисляем secret_key = HMAC_SHA256(bot_token, "WebAppData")
	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))

	// Вычисляем hash = HMAC_SHA256(secret_key, data_check_string)
	h := hmac.New(sha256.New, secretKey.Sum(nil))
	h.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(h.Sum(nil))

	if calculatedHash != hash {
		return fmt.Errorf("неверная подпись данных")
	}

	return nil
}
