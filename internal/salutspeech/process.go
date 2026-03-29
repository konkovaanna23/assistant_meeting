package salutspeech

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"
)

var endStatuses = map[string]bool{
	"CANCELED": true,
	"DONE":     true,
	"ERROR":    true,
}

type Task struct {
	PathFile       string
	FileID         string
	TaskID         string
	Status         string
	Done           chan struct{}
	Result         string
	ErrorMessage   string
	ResponseFileID string
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
			ss.waitResult(ctx, task)
		}
	}
}

func (ss *SalutSpeechClient) waitResult(ctx context.Context, task *Task) {
	defer close(task.Done)
end:
	for {
		select {
		case <-ctx.Done():
			ss.lgr.Info("Выполнение прервано, статус по задаче не будет получен", zap.String("taskID", task.TaskID))
			return
		default:
			taskStatus, responseFileID, err := ss.GetStatusTask(task.TaskID)

			if err != nil {
				ss.lgr.Error("ошибка получения статуса", zap.Error(err))
			} else {
				ss.lgr.Debug("Получен статус задачи", zap.String("taskID", task.TaskID), zap.String("status", taskStatus))
				task.Status = taskStatus
				task.ResponseFileID = responseFileID
				if endStatuses[taskStatus] {
					break end
				}
			}
			time.Sleep(time.Duration(2) * time.Second) /*TODO в конфиги*/
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

func (ss *SalutSpeechClient) RecognizeFile(ctx context.Context, pathFile string) (*Task, error) {
	ss.lgr.Info("Запрос на распознавание файла", zap.String("file", pathFile))
	requestFileID, err := ss.UploadFile(pathFile)
	if err != nil {
		log.Printf("ошибка загрузки файла %s", err.Error())
		return nil, err
	}

	ss.lgr.Debug("Файл загружен", zap.String("fileID", requestFileID))

	taskStatus, err := ss.CreateTaskRecognize(requestFileID)
	if err != nil {
		log.Printf("ошибка создания задачи на распознование %s", err.Error())
		return nil, err
	}

	ss.lgr.Debug("Установлена задача на распознавание", zap.String("taskID", taskStatus.ID))

	task := &Task{FileID: requestFileID,
		PathFile:       pathFile,
		TaskID:         taskStatus.ID,
		Status:         taskStatus.Status,
		Done:           make(chan struct{}),
		ResponseFileID: taskStatus.ResponseFileID,
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
