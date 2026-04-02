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

var insertAudio string = `INSERT INTO users_audio(id, user_id, path) values($1,$2,$3)`

var updateStatusAudio string = `update users_audio 
								set  file_id=$2,task_id=$3,status=$4,updated_at=NOW()
								where id=$1`

var updateShortTextAudio string = `update users_audio 
								set  result_short=$2,updated_at=NOW()
								where id=$1`

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

func (ds *DBStore) CreateAudio(ctx context.Context, userID int64, file string) (string, error) {
	id := model.NewUUID()

	_, err := ds.db.ExecContext(ctx, insertAudio, id, userID, file)
	if err != nil {
		return "", fmt.Errorf("ошибка сохранения аудио: %w", err)
	}

	return id, nil
}

func (ds *DBStore) UpdateStatusTask(ctx context.Context, id, taskID, fileID, status string) error {

	_, err := ds.db.ExecContext(ctx, updateStatusAudio, id, fileID, taskID, status)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса аудио: %w", err)
	}

	return nil
}

/*func (ds *DBStore) UpdateResult(ctx context.Context, userID int64, file string) (string, error) {
	id := model.NewUUID()

	_, err := ds.db.ExecContext(ctx, insertAudio, id, userID, file)
	if err != nil {
		return "", fmt.Errorf("ошибка сохранения аудио: %w", err)
	}

	return id, nil
}*/

func (ds *DBStore) UpdateShortText(ctx context.Context, id int64, text string) error {

	_, err := ds.db.ExecContext(ctx, updateShortTextAudio, id, text)
	if err != nil {
		return fmt.Errorf("ошибка обновления краткой выжимки аудио: %w", err)
	}

	return nil
}
