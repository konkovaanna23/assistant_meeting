package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/konkovaanna23/assistant_meeting/internal/model"
)

var selectAudioByID string = `SELECT result_short FROM users_audio WHERE user_id = $1 and id=$2`

var selectAudioByWord string = `SELECT a.id, a.path FROM users_audio a inner join users_audio_words w on w.id=a.id WHERE a.user_id = $1 and word=$2`

var ErrorNotContent = errors.New("data not found")

func (ds *DBStore) GetAudioByID(ctx context.Context, userID int64, audioID string) (string, error) {
	var text string

	err := ds.db.QueryRowxContext(ctx, selectAudioByID, userID, audioID).Scan(&text)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrorNotContent
		}
		return "", fmt.Errorf("ошибка получения текст аудио из БД: %w", err)
	}

	return text, nil
}

func (ds *DBStore) GetAudioByWord(ctx context.Context, userID int64, word string) ([]*model.AudioShort, error) {
	var audioList []*model.AudioShort

	rows, err := ds.db.QueryContext(ctx, selectAudioByWord, userID)
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
