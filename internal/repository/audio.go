package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/konkovaanna23/assistant_meeting/internal/model"
)

var ErrorNotContent = errors.New("data not found")

var selectAudioByID string = `SELECT result_short, result_text, is_voice FROM recognition.users_audio WHERE user_id = $1 and id=$2`

var selectAudioByWord string = `SELECT a.id, a.path FROM recognition.users_audio a inner join recognition.users_audio_words w on w.id=a.id WHERE a.user_id = $1 and word=$2 `

var insertAudio string = `INSERT INTO recognition.users_audio(id, user_id, path, is_voice) values($1,$2,$3,$4)`

var updateStatusAudio string = `update recognition.users_audio 
								set  file_id=$2,task_id=$3,status=$4,updated_at=NOW()
								where id=$1`

var updateShortTextAudio string = `update recognition.users_audio 
								set  result_short=$2,updated_at=NOW()
								where id=$1`

var updateResult string = `update recognition.users_audio 
								set  result=$2, result_text=$3, updated_at=NOW()
								where id=$1`

var insertWords string = `
				INSERT INTO recognition.users_audio_words (id, word)
				VALUES %s
			`

func (ds *DBStore) GetAudioByID(ctx context.Context, userID int64, audioID string) (string, string, bool, error) {
	var text, shortText string
	var isVoice bool

	err := ds.db.QueryRowxContext(ctx, selectAudioByID, userID, audioID).Scan(&shortText, &text, &isVoice)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", false, ErrorNotContent
		}
		return "", "", false, fmt.Errorf("ошибка получения текст аудио из БД: %w", err)
	}

	return shortText, text, isVoice, nil
}

func (ds *DBStore) GetAudioByWord(ctx context.Context, userID int64, word string) ([]*model.AudioShort, error) {
	var audioList []*model.AudioShort

	rows, err := ds.db.QueryContext(ctx, selectAudioByWord, userID, word)
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

func (ds *DBStore) CreateAudio(ctx context.Context, userID int64, file string, isVoice bool) (string, error) {
	id := model.NewUUID()

	_, err := ds.db.ExecContext(ctx, insertAudio, id, userID, file, isVoice)
	if err != nil {
		return "", fmt.Errorf("ошибка сохранения аудио: %w", err)
	}

	return id, nil
}

func (ds *DBStore) UpdateStatusTask(ctx context.Context, id, taskID, fileID, status string) error {

	fmt.Printf("id %s fileID %s taskID %s", id, taskID, fileID)
	_, err := ds.db.ExecContext(ctx, updateStatusAudio, id, toNullString(fileID), toNullString(taskID), status)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса аудио: %w", err)
	}

	return nil
}

func (ds *DBStore) UpdateShortText(ctx context.Context, id string, text string) error {

	_, err := ds.db.ExecContext(ctx, updateShortTextAudio, id, text)
	if err != nil {
		return fmt.Errorf("ошибка обновления краткой выжимки аудио: %w", err)
	}

	return nil
}

func (ds *DBStore) SaveResult(ctx context.Context, id string, resultJson string, result *model.CombinedResult) error {
	tx, err := ds.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.ExecContext(ctx, updateResult, id, resultJson, result.NormalizedText)
	if err != nil {
		return fmt.Errorf("обновление результата выполнено с ошибкой: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("row with id=%d not found", id)
	}

	if len(result.WordAlignments) > 0 {
		valueParts := make([]string, 0, len(result.WordAlignments))
		args := make([]any, 0, len(result.WordAlignments)*2)

		for _, wa := range result.WordAlignments {
			word := strings.TrimSpace(wa.Word)
			if word == "" {
				continue
			}

			valueParts = append(valueParts, "(?, ?)")
			args = append(args, id, word)
		}

		if len(valueParts) > 0 {
			insertQuery := fmt.Sprintf(insertWords, strings.Join(valueParts, ","))
			insertQuery = tx.Rebind(insertQuery)

			if _, err = tx.ExecContext(ctx, insertQuery, args...); err != nil {
				return fmt.Errorf("insert words: %w", err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func toNullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}
