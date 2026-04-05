package salutspeech

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/konkovaanna23/assistant_meeting/internal/model"
	"go.uber.org/zap"
)

var endStatuses = map[string]bool{
	"CANCELED": true,
	"DONE":     true,
	"ERROR":    true,
}

func isDone(status string) bool {
	return status == "DONE"
}

func (ss *SalutSpeechClient) StartWorkers(ctx context.Context, n int) {
	for i := 0; i < n; i++ {
		go ss.worker(ctx, i+1)
	}
}

func (ss *SalutSpeechClient) worker(ctx context.Context, i int) {
	log.Printf("worker %d запущен", i)
	for {
		select {
		case <-ctx.Done():
			ss.lgr.Info("Worker прерван", zap.Int("worker", i))
			return
		case task := <-ss.requests:
			ss.lgr.Debug("Запрос на ожидание статуса", zap.String("taskID", task.TaskID))
			if task.CtxTask.Err() == nil {
				ss.waitResult(ctx, task)
			}

		}
	}
}

func (ss *SalutSpeechClient) waitResult(ctx context.Context, task *model.Task) {
	defer close(task.Done)
	previousTaskStatus := task.Status
end:
	for {
		select {
		case <-ctx.Done():
			ss.lgr.Info("Выполнение прервано, статус по задаче не будет получен", zap.String("taskID", task.TaskID))
			return
		case <-task.CtxTask.Done():
			ss.lgr.Info("Выполнение прервано, статус по задаче не будет получен", zap.String("taskID", task.TaskID))
			return
		default:
			taskStatus, responseFileID, errMessage, err := ss.GetStatusTask(task.TaskID)
			if err != nil {
				ss.lgr.Error("ошибка получения статуса", zap.Error(err))
			} else {
				if taskStatus != previousTaskStatus {
					err := task.FunsSaveStatus(task.CtxTask, task.AudioID, task.TaskID, task.FileID, taskStatus)
					if err != nil {
						ss.lgr.Debug("Ошибка сохранения статус задачи", zap.String("AudioID", task.AudioID), zap.String("taskID", task.TaskID), zap.String("status", taskStatus), zap.Error(err))
					}
					previousTaskStatus = taskStatus
				}
				ss.lgr.Debug("Получен статус задачи", zap.String("taskID", task.TaskID), zap.String("status", taskStatus), zap.String("message_error", errMessage))
				task.Status = taskStatus
				task.ResponseFileID = responseFileID
				if endStatuses[taskStatus] {
					break end
				}
			}
			time.Sleep(ss.periodPolling)
		}

	}

	if isDone(task.Status) {
		ss.lgr.Info("Запрос данных по файлу", zap.String("responseFileID", task.ResponseFileID))
		data, err := ss.GetData(task.ResponseFileID)
		if err != nil {
			errMsg := fmt.Sprintf("ошибка получения информации %s", err.Error())
			ss.lgr.Error("ошибка получения информации ", zap.Error(err))
			task.ErrorMessage = errMsg
		}
		task.Result = data
	} else {
		errMsg := fmt.Sprintf("задача завершилась со статусом %s", task.Status)
		ss.lgr.Error("задача завершилась со статусом ", zap.String("status", task.Status))
		task.ErrorMessage = errMsg
	}

}

func (ss *SalutSpeechClient) RecognizeFile(ctx context.Context, inputAudio *model.InputAudio, funsSaveStatus func(ctx context.Context, id, taskID, fileID, status string) error) (*model.Task, error) {
	ss.lgr.Info("Запрос на распознавание файла", zap.String("file", inputAudio.FileName))

	audioSpec, err := DetectAudioSpec(inputAudio)
	if err != nil {
		return nil, err
	}
	requestFileID, err := ss.UploadFile(inputAudio.FileName, audioSpec.ContentType)
	if err != nil {
		log.Printf("ошибка загрузки файла %s", err.Error())
		return nil, err
	}

	ss.lgr.Debug("Файл загружен", zap.String("fileID", requestFileID))

	taskStatus, err := ss.CreateTaskRecognize(requestFileID, audioSpec.Encoding, audioSpec.Channels, audioSpec.SampleRate)
	if err != nil {
		log.Printf("ошибка создания задачи на распознование %s", err.Error())
		return nil, err
	}

	ss.lgr.Debug("Установлена задача на распознавание", zap.String("taskID", taskStatus.ID))

	task := &model.Task{FileID: requestFileID,
		PathFile:       inputAudio.FileName,
		TaskID:         taskStatus.ID,
		Status:         taskStatus.Status,
		Done:           make(chan struct{}),
		ResponseFileID: taskStatus.ResponseFileID,
		FunsSaveStatus: funsSaveStatus,
		CtxTask:        ctx,
		AudioID:        inputAudio.AudioID,
	}

	err = task.FunsSaveStatus(ctx, inputAudio.AudioID, taskStatus.ID, requestFileID, taskStatus.Status)
	if err != nil {
		ss.lgr.Debug("Ошибка сохранения статус задачи", zap.String("AudioID", task.AudioID), zap.String("taskID", task.TaskID), zap.String("status", taskStatus.Status), zap.Error(err))
	}
	ss.lgr.Debug("Получен статус", zap.String("taskID", taskStatus.ID), zap.String("status", taskStatus.Status))

	if endStatuses[task.Status] {
		if isDone(task.Status) {

			ss.lgr.Debug("Получен конечный успешный статус, запрашиваем данные", zap.String("status", taskStatus.Status), zap.String("responceFileID", taskStatus.ResponseFileID))

			data, err := ss.GetData(taskStatus.ResponseFileID)
			if err != nil {
				return nil, fmt.Errorf("ошибка при получении данны %s", err.Error())
			}
			task.Result = data
		}
		close(task.Done)
		return task, nil
	}

	select {
	case ss.requests <- task:
		return task, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}

}
