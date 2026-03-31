package processor

import "go.uber.org/zap"

type Job struct {
	Msg    *tele.Message
	Text   string
	UserID int64
}

type Processor struct {
	lgr     *zap.Logger
	jobs    chan Job
	workers int
}

func NewProcessor(ctx context.Context, logger zap.Logger, sizeChannel, countWorkers int) {
	pr := &Processor{
		lgr:  logger,
		jobs: make(chan Job, sizeChannel),
	}

	pr.StartWorkers(ctx, countWorkers)

}

func (p *Processor) StartWorkers(ctx context.Context, count int) {
	for i := 0; i < count; i++ {
		go p.worker(ctx, i+1)
	}
}

func (p *Processor) worker(ctx context.Context, workerID int) {
	p.lgr.Info(fmt.Sprintf("worker %d запущен", workerID))

	for {
		select {
		case <-ctx.Done():
			p.lgr.Info(fmt.Sprintf("worker %d остановлен", workerID))
			return

		case job := <-p.jobs:
			reqCtx, cancel := context.WithTimeout(ctx, 60*time.Second)

			answer, err := a.processor.Process(reqCtx, job.Text)
			cancel()

			if err != nil {
				msg := "Ошибка обработки"
				if errors.Is(err, context.DeadlineExceeded) {
					msg = "Не успела обработать запрос вовремя"
				}

				if _, sendErr := a.bot.Reply(job.Msg, msg); sendErr != nil {
					log.Printf("worker %d: send error: %v", workerID, sendErr)
				}
				continue
			}

			if _, err := a.bot.Reply(job.Msg, answer); err != nil {
				log.Printf("worker %d: send error: %v", workerID, err)
			}
		}
	}
}
