package repository

import (
	"context"
	"fmt"

	"github.com/konkovaanna23/assistant_meeting/internal/model"
)

var insertUser string = `
		INSERT INTO recognition.users (id, chat_id, username)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING;
	`
var selectAudioForUser string = `SELECT id, path FROM recognition.users_audio WHERE user_id = $1 and status='DONE' and result_short is not null`

// CreateUser - добавление в users.
func (ds *DBStore) CreateUser(ctx context.Context, id, chatid int64, username string) error {

	_, err := ds.db.ExecContext(ctx, insertUser, id, chatid, username)
	if err != nil {
		return fmt.Errorf("ошибка вставки в таблицу users: %w", err)
	}

	return nil
}

// GetAudioListForUser -
func (ds *DBStore) GetAudioListForUser(ctx context.Context, userID int64) ([]*model.AudioShort, error) {
	var audioList []*model.AudioShort

	rows, err := ds.db.QueryContext(ctx, selectAudioForUser, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var audio model.AudioShort
		err := rows.Scan(
			&audio.ID,
			&audio.Path,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}
		audioList = append(audioList, &audio)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при чтении строк: %w", err)
	}

	return audioList, nil
}
