package salutspeech

import (
	"context"
	"log"
)

var endStatuses = map[string]bool{
	"CANCELED": true,
	"DONE":     true,
	"ERROR":    true,
}

type Task struct {
	FileID string
	TaskID string
	Status string
	Done   chan struct{}
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
		case pathFileNew := <-ss.requests:
			ss.processFile(ctx, pathFileNew)
		}
	}
}

func (ss *SalutSpeechClient) monitorStatus(ctx context.Context, taskID string) {

end:
	for {
		select {
		case <-ctx.Done():
			log.Printf("отмена контекста, задача %s не будет выполнена")

		default:
			taskStatus, err := ss.GetStatusTask(taskID)
			if err != nil {
				log.Printf("ошибка получения статуса %s", err.Error())
			}
			if endStatuses[taskStatus] {
				break end
			}
		}

	}

	data, err := ss.GetData()

}

func (ss *SalutSpeechClient) RecognizeFile(ctx context.Context, pathFileNew string) error {
	requestFileID, err := ss.UploadFile(pathFileNew)
	if err != nil {
		log.Printf("ошибка загрузки файла %s", err.Error())
		return err
	}

	taskStatus, err := ss.CreateTaskRecognize(requestFileID)
	if err != nil {
		log.Printf("ошибка создания задачи на распознование %s", err.Error())
		return err
	}

	select {
	case ss.requests <- taskStatus.ID:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}

}
