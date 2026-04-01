// package repository
package repository

import (
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// DBStore хранилище репозиториев.
type DBStore struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewDBStore(logger *zap.Logger, db *sqlx.DB) *DBStore {
	return &DBStore{
		db:     db,
		logger: logger,
	}
}
