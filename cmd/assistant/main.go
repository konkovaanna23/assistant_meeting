package main

import (
	"context"
	"github.com/konkovaanna23/assistant_meeting/internal/config"
	"github.com/konkovaanna23/assistant_meeting/internal/config/db"
	gg "github.com/konkovaanna23/assistant_meeting/internal/gigachat"
	"github.com/konkovaanna23/assistant_meeting/internal/processor"
	"github.com/konkovaanna23/assistant_meeting/internal/repository"
	ss "github.com/konkovaanna23/assistant_meeting/internal/salutspeech"
	"go.uber.org/zap"
	"log"
	"os/signal"
	"syscall"
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

	gigaChat, err := gg.NewGigaChatClient(ctx, cfg.GigaChat, logger.With(zap.String("component", "gigachat")))
	if err != nil {
		logger.Fatal("Ошибка создания gigachat", zap.Error(err))
	}

	database, err := db.NewConnect(cfg.DSN)
	if err != nil {
		logger.Fatal("ошибка при подключении к базе данных:", zap.Error(err))
	} else {
		logger.Info("подключение к базе данных успешно")
		if err := db.RunMigrations(cfg.DSN); err != nil {
			logger.Fatal("ошибка при установке миграций:", zap.Error(err))
		}
	}

	repo := repository.NewDBStore(logger.With(zap.String("component", "repository")), database)

	processorBot, err := processor.NewProcessorBot(logger.With(zap.String("component", "processor")),
		cfg.SizeChannel,
		cfg.CountWorkers,
		cfg.TelegramToken,
		cfg.PollingBot,
		salutSpeech,
		gigaChat,
		repo)

	if err != nil {
		logger.Fatal("Ошибка создания telegram bot", zap.Error(err))
	}

	logger.Info("Запуск AssistantMeeting...")
	go processorBot.Start(ctx)

	<-ctx.Done()

	processorBot.Stop()
	logger.Info("Сервер остановлен")
}
