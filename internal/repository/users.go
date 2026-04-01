package repository

import (
	"context"
	"fmt"
)

var insertUser string = `
		INSERT INTO users (id, chat_id, username)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING;
	`
var selectUserLogin string = `SELECT id, password FROM storage.users WHERE login = $1`

// CreateUser - добавление в users.
func (ds *DBStore) CreateUser(ctx context.Context, id, chatid int64, username string) error {

	_, err := ds.db.ExecContext(ctx, insertUser, id, chatid, username)
	if err != nil {
		return fmt.Errorf("ошибка вставки в таблицу users: %w", err)
	}

	return nil
}

/*
// GetUserByLogin - получение пользователя по логину.
func (ds *userStore) GetUserByLogin(ctx context.Context, loginSrc string) (string, string, error) {
	var id, password string

	err := ds.database.QueryRowxContext(ctx, selectUserLogin, loginSrc).Scan(&id, &password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrorUserNotFound
		}
		return "", "", fmt.Errorf("ошибка получения пользователя из БД: %w", err)
	}

	return id, password, nil
}
*/
