package postgres

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
)

type DB struct {
	Conn *pgx.Conn
}

func NewPostgresDB(connString string) (*DB, error) {
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к PostgreSQL: %w", err)
	}

	if err = conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("ошибка пинга PostgreSQL: %w", err)
	}

	db := &DB{Conn: conn}

	// Запускаем миграции
	if err := runMigrations(connString); err != nil {
		return nil, fmt.Errorf("ошибка миграций: %w", err)
	}

	log.Println("✅ Подключение к БД успешно, миграции применены!")
	return db, nil
}

func runMigrations(connString string) error {
	// Драйвер миграций ожидает префикс postgres:// или postgresql://
	// Но pgx понимает оба, на всякий случай убедимся, что формат верный
	if !strings.HasPrefix(connString, "postgres://") && !strings.HasPrefix(connString, "postgresql://") {
		return fmt.Errorf("строка подключения должна начинаться с postgres:// или postgresql://")
	}

	// Указываем путь к папке с миграциями (file://migrations)
	m, err := migrate.New("file://migrations", connString)
	if err != nil {
		return fmt.Errorf("ошибка инициализации migrate: %w", err)
	}

	// Пытаемся накатить миграции
	err = m.Up()
	if err != nil {
		if err == migrate.ErrNoChange {
			log.Println("ℹ️ Структура БД актуальна, новых миграций нет.")
			return nil
		}
		return err
	}

	log.Println("🔥 Миграции успешно применены к базе данных!")
	return nil
}
