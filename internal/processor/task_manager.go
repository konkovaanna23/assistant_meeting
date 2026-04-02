package processor

import (
	"context"
	"errors"

	"github.com/gogo/protobuf/plugin/stringer"
	"github.com/konkovaanna23/assistant_meeting/internal/model"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/result"
	"go.uber.org/zap"
)

func (p *ProcessorBot) process(ctx context.Context, job *Job) (string, error) {
	switch job.Handler {
	case "/start":
		err := p.repo.CreateUser(ctx, job.UserID, job.ChatID, job.Username)
		if err != nil {
			return "", err
		}
		return "", nil
	case "/get":
		result, err := p.repo.GetAudioByID(ctx, job.UserID, job.IdAudio)
		if err != nil {
			return "", err
		}
		return result, nil
	case "/list":
		result, err := p.repo.GetAudioListForUser(ctx, job.UserID)
		if err != nil {
			return "", err
		}
		return model.FormatAudioList(result), nil
	case "/find":
		result, err := p.repo.GetAudioByWord(ctx, job.UserID, job.KeyWord)
		if err != nil {
			return "", err
		}
		return model.FormatAudioList(result), nil
	case "OnVoice", "OnAudio":
		result, err := p.processAudio(ctx, job.UserID, job.Handler == "OnVoice", job.FileName, job.FilePath, job.MIME)
		if err != nil {
			return "", err
		}
		return result, nil
	default:
		return "", errors.New("неизвестная команда")
	}
}

func (p *ProcessorBot) processAudio(ctx context.Context, userID int64, isVoice bool, sourceFileName string, filePath string, mime string) (string, error) {
	id, err := p.repo.CreateAudio(ctx, userID, sourceFileName)
	if err != nil {
		p.lgr.Error("Ошибка создания ", zap.Int64("userID", userID), zap.String("fileName", sourceFileName), zap.Error(err))
		return "", errors.New("ошибка распознавания файла")
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
		return "", errors.New("ошибка распознавания файла")
	}
	<-task.Done

	p.saveResultAudio(id, task.Result)

	result, err := p.gigachat.GetBriefExtract(task.Result)
	if err != nil {
		p.lgr.Error("ошибка получения краткой выжимки", zap.Int64("userID", userID), zap.String("fileName", sourceFileName), zap.Error(err))

		return "", errors.New("ошибка получения краткой выжимки")
	}

	p.repo.UpdateShortText(ctx context.Context, id, result)

	return result, nil
}
