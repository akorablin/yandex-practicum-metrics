package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

func Init(databaseDSN string) error {
	// Подключение к БД
	var err error
	DB, err = sql.Open("pgx", databaseDSN)
	if err != nil {
		return fmt.Errorf("Не удалось подключиться к БД: %v", err)
	}
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("Проверка подключения к БД завершилаь с ошибкой: %v", err)
	}

	// Запуск миграций
	driver, err := postgres.WithInstance(DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("Ошибка создания драйвера миграций: %v", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("Ошибка создания миграции: %v", err)
	}
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("Ошибка применения миграций: %v", err)
	}
	log.Println("Миграции успешно применены!")

	return nil
}

func GetDB() *sql.DB {
	return DB
}
