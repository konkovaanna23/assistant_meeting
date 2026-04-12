// Package db - пакет для подключения к базе данных.
package db

import (
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// NewConnect создает подключение к базе данных.
func NewConnect(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := Ping(db); err != nil {
		return nil, err
	}

	return db, nil
}

// Ping проверяет подключение к базе данных.
func Ping(db *sqlx.DB) error {
	if err := db.Ping(); err != nil {
		return err
	}
	return nil
}

// RunMigrations запускает миграции базы данных.
func RunMigrations(dsn string) error {
	m, err := migrate.New(
		"file://migrations",
		dsn,
	)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	logrus.Println("Миграции применены")
	return nil
}
