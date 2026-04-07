package processor

import (
	"context"
	"errors"
	"fmt"

	"github.com/konkovaanna23/assistant_meeting/internal/model"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (p *ProcessorBot) process(ctx context.Context, job *Job) (string, []interface{}, error) {
	switch job.Handler {
	case "/start":
		err := p.repo.CreateUser(ctx, job.UserID, job.ChatID, job.Username)
		if err != nil {
			return "", nil, err
		}
		return "Привет! AssistantMeeting готов к работе", nil, nil
	case "/get":
		resultShort, result, isVoice, err := p.repo.GetAudioByID(ctx, job.UserID, job.IdAudio)
		if err != nil {
			return "", nil, err
		}

		p.saveSessionRequest(job.UserID, model.RoleUser, p.gigachat.GetTextRequest(result, isVoice))
		p.saveSessionRequest(job.UserID, model.RoleAssistant, resultShort)

		return resultShort, nil, nil
	case "/list":
		result, err := p.listAudio(ctx, job.UserID)
		if err != nil {
			return "", nil, err
		}
		return "Выбери аудио:", []interface{}{result}, nil
	case "/find":
		result, err := p.repo.GetAudioByWord(ctx, job.UserID, job.KeyWord)
		if err != nil {
			return "", nil, err
		}
		if len(result) == 0 {
			return fmt.Sprintf("По слову [%s] не найдено аудио", job.KeyWord), nil, nil
		}
		return model.FormatAudioList(result), nil, nil
	case "OnVoice", "OnAudio":
		result, resultSource, err := p.processAudio(ctx, job.UserID, job.Handler == "OnVoice", job.FileName, job.FilePath, job.MIME)
		if err != nil {
			return "", nil, err
		}

		p.saveSessionRequest(job.UserID, model.RoleUser, p.gigachat.GetTextRequest(resultSource, job.Handler == "OnVoice"))
		p.saveSessionRequest(job.UserID, model.RoleAssistant, result)

		return result, nil, nil

	case "OnText":
		p.saveSessionRequest(job.UserID, model.RoleUser, job.Text)

		messages := p.GetSession(job.UserID)

		result, err := p.gigachat.Chat(ctx, messages)
		if err != nil {
			return "", nil, err
		}

		p.saveSessionRequest(job.UserID, model.RoleAssistant, result)

		return result, nil, nil

	default:
		return "", nil, errors.New("неизвестная команда")
	}
}

func (p *ProcessorBot) listAudio(ctx context.Context, userID int64) (*tele.ReplyMarkup, error) {
	result, err := p.repo.GetAudioListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	markup := &tele.ReplyMarkup{}

	var rows []tele.Row
	for _, item := range result {
		btn := markup.Data(
			fmt.Sprintf("%s (%s)", item.Path, item.ID),
			"item_get",
			item.ID,
		)
		rows = append(rows, markup.Row(btn))
	}

	markup.Inline(rows...)

	return markup, nil
}

func (p *ProcessorBot) processAudio(ctx context.Context, userID int64, isVoice bool, sourceFileName string, filePath string, mime string) (string, string, error) {
	id, err := p.repo.CreateAudio(ctx, userID, sourceFileName, isVoice)
	if err != nil {
		p.lgr.Error("Ошибка создания ", zap.Int64("userID", userID), zap.String("fileName", sourceFileName), zap.Error(err))
		return "", "", errors.New("ошибка распознавания файла")
	}
	in := &model.InputAudio{
		AudioID:  id,
		FileName: filePath,
		MIME:     mime,
		IsVoice:  isVoice,
	}

	task, err := p.salutSpeech.RecognizeFile(ctx, in, p.repo.UpdateStatusTask)
	if err != nil {
		p.lgr.Error("Ошибка распознавания", zap.Int64("userID", userID), zap.String("fileName", sourceFileName), zap.Error(err))
		return "", "", errors.New("ошибка распознавания файла")
	}
	<-task.Done

	text, err := p.saveResultAudio(ctx, id, task.Result)
	if err != nil {
		p.lgr.Error("Ошибка сохранения результата", zap.Int64("userID", userID), zap.String("fileName", sourceFileName), zap.Error(err))
		return "", "", errors.New("ошибка распознавания файла")
	}

	result, err := p.gigachat.GetBriefExtract(text, isVoice)
	if err != nil {
		p.lgr.Error("ошибка получения краткой выжимки", zap.Int64("userID", userID), zap.String("fileName", sourceFileName), zap.Error(err))
		return "", "", errors.New("ошибка получения краткой выжимки")
	}

	p.repo.UpdateShortText(ctx, id, result)

	return result, task.Result, nil
}

func (p *ProcessorBot) saveResultAudio(ctx context.Context, id string, resultJson string) (string, error) {
	result, err := BuildCombinedJSON(resultJson)
	if err != nil {
		p.lgr.Error("Ошибка парсинга результата", zap.String("AudioID", id), zap.Error(err))

		return "", fmt.Errorf("ошибка парсинга результата %s", err.Error())
	}
	err = p.repo.SaveResult(ctx, id, resultJson, result)
	if err != nil {
		p.lgr.Error("Ошибка сохранения результата", zap.String("AudioID", id), zap.Error(err))
		return "", fmt.Errorf("ошибка сохранения результата %s", err.Error())
	}
	return result.NormalizedText, nil
}
