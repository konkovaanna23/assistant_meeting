package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/konkovaanna23/assistant_meeting/internal/config"
	gg "github.com/konkovaanna23/assistant_meeting/internal/gigachat"
	ss "github.com/konkovaanna23/assistant_meeting/internal/salutspeech"
	"github.com/konkovaanna23/assistant_meeting/internal/telegram"
	"go.uber.org/zap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	cfg, err := config.GetConfig()
	if err != nil {
		logger.Fatal("Ошибка чтения конфига:", zap.Error(err))
	}

	salutSpeech, err := ss.NewSalutSpeechClient(ctx, cfg.SalutSpeech, cfg.CountWorkers, cfg.SizeChannel, cfg.PeriodPolling, logger.With(zap.String("component", "salutspeech")))
	if err != nil {
		logger.Fatal("Ошибка создания salut speech", zap.Error(err))
	}

	gigaChat, err := gg.NewGigaChatClient(ctx, cfg.SalutSpeech, logger.With(zap.String("component", "gigachat")))
	if err != nil {
		logger.Fatal("Ошибка создания gigachat", zap.Error(err))
	}

	tgBot, err := telegram.NewTelegramBot(logger, cfg.TelegramToken, 10, salutSpeech, gigaChat)
	if err != nil {
		logger.Fatal("Ошибка создания telegram", zap.Error(err))
	}

	logger.Info("Запуск AssistantMeeting...")
	go tgBot.Start()

	<-ctx.Done()

	tgBot.Stop()
	logger.Info("Сервер остановлен")
}
