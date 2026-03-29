package salutspeech

import (
	"context"
	"fmt"
	"log"
	"time"
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
			log.Printf("worker %d прерван", i)
		case task := <-ss.requests:
			ss.lgr.Debug()
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
			log.Printf("отмена контекста, задача %s не будет выполнена")

		default:
			taskStatus, err := ss.GetStatusTask(task.TaskID)
			if err != nil {
				log.Printf("ошибка получения статуса %s", err.Error())
			} else {
				task.Status = taskStatus
				if endStatuses[taskStatus] {
					task.Status = taskStatus
					break end
				}
			}
			time.Sleep(time.Duration(2) * time.Second) /*TODO в конфиги*/
		}

	}

	if isDone(task.Status) {
		data, err := ss.GetData(task.FileID)
		if err != nil {
			errMsg := fmt.Sprintf("ошибка получения информации %s", err.Error())
			task.ErrorMessage = errMsg
		}
		task.Result = data
	} else {
		errMsg := fmt.Sprintf("задача завершилась со статусом %s", task.Status)
		task.ErrorMessage = errMsg
	}

}

func (ss *SalutSpeechClient) RecognizeFile(ctx context.Context, pathFile string) (*Task, error) {
	requestFileID, err := ss.UploadFile(pathFile)
	if err != nil {
		log.Printf("ошибка загрузки файла %s", err.Error())
		return nil, err
	}

	taskStatus, err := ss.CreateTaskRecognize(requestFileID)
	if err != nil {
		log.Printf("ошибка создания задачи на распознование %s", err.Error())
		return nil, err
	}

	task := &Task{FileID: requestFileID,
		PathFile: pathFile,
		TaskID:   taskStatus.ID,
		Status:   taskStatus.Status,
	}
	if endStatuses[task.Status] {
		if isDone(task.Status) {
			data, err := ss.GetData(requestFileID)
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
