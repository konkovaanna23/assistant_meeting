package processor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/konkovaanna23/assistant_meeting/internal/gigachat"
	"github.com/konkovaanna23/assistant_meeting/internal/repository"
	"github.com/konkovaanna23/assistant_meeting/internal/salutspeech"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

type Job struct {
	Msg      *tele.Message
	Text     string
	UserID   int64
	Username string
	ChatID   int64

	IdAudio  string
	KeyWord  string
	Handler  string
	FilePath string
	MIME     string
	FileName string
}

type ProcessorBot struct {
	lgr     *zap.Logger
	jobs    chan Job
	workers int

	bot *tele.Bot

	salutSpeech *salutspeech.SalutSpeechClient
	gigachat    *gigachat.GigaChatClient

	repo *repository.DBStore
}

func NewProcessorBot(logger *zap.Logger, sizeChannel, countWorkers int, tokenBot string, pollingPeriodBot int, ss *salutspeech.SalutSpeechClient, gg *gigachat.GigaChatClient, repository *repository.DBStore) (*ProcessorBot, error) {

	pr := &ProcessorBot{
		lgr:         logger,
		jobs:        make(chan Job, sizeChannel),
		workers:     countWorkers,
		salutSpeech: ss,
		gigachat:    gg,
		repo:        repository,
	}
	bot, err := pr.newTeleBot(tokenBot, pollingPeriodBot)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания подключения к telegram bot: %s", err.Error())
	}

	pr.bot = bot

	err = ensureUploadDir("./" + dirDownloads)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать директорию для загрузки audio %s", err.Error())
	}

	return pr, nil
}

func (p *ProcessorBot) newTeleBot(token string, pollingPeriod int) (*tele.Bot, error) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: time.Duration(pollingPeriod) * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	b.Handle("/start", p.SaveUser)
	b.Handle("/list", p.GetListAudio)
	b.Handle("/get", p.GetTextAudio)
	b.Handle("/chat", p.GigaChatRequest)
	b.Handle("/find", p.FindAudio)

	b.Handle(tele.OnText, p.handlerOnText)

	b.Handle(tele.OnAudio, p.handlerOnAudio)

	b.Handle(tele.OnVoice, p.hanlerOnVoice)

	return b, nil

}

func (p *ProcessorBot) Start(ctx context.Context) {

	p.startWorkers(ctx)

	p.bot.Start()

}

func (p *ProcessorBot) Stop() {
	p.bot.Stop()
}

func (p *ProcessorBot) startWorkers(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		go p.worker(ctx, i+1)
	}
}

func (p *ProcessorBot) worker(ctx context.Context, workerID int) {
	p.lgr.Info(fmt.Sprintf("worker %d запущен", workerID))

	for {
		select {
		case <-ctx.Done():
			p.lgr.Info(fmt.Sprintf("worker %d остановлен", workerID))
			return

		case job := <-p.jobs:
			reqCtx, cancel := context.WithTimeout(ctx, 60*time.Second) /*TODO конфиги*/

			answer, err := p.process(reqCtx, &job)
			cancel()

			chat := &tele.Chat{ID: job.ChatID}

			if err != nil {
				msg := "Ошибка обработки"
				if errors.Is(err, context.DeadlineExceeded) {
					msg = "Не успела обработать запрос вовремя"
				}

				chat := &tele.Chat{ID: job.ChatID}

				p.lgr.Error("ошибка в работе handler", zap.Int("worker", workerID), zap.String("handler", job.Handler), zap.Int64("userID", job.UserID), zap.Error(err))

				if _, sendErr := p.bot.Send(chat, msg); sendErr != nil {
					p.lgr.Error("ошибка отправки", zap.Int("worker", workerID), zap.Error(sendErr))
				}
				continue
			}

			if _, sendErr := p.bot.Send(chat, answer); sendErr != nil {
				p.lgr.Error("ошибка отправки", zap.Int("worker", workerID), zap.String("handler", job.Handler), zap.Int64("userID", job.UserID), zap.Error(sendErr))
			}
		}
	}
}
