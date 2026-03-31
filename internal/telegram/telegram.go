package telegram

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/konkovaanna23/assistant_meeting/internal/gigachat"
	"github.com/konkovaanna23/assistant_meeting/internal/model"
	"github.com/konkovaanna23/assistant_meeting/internal/salutspeech"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

type TelegramBot struct {
	bot *tele.Bot

	salutSpeech *salutspeech.SalutSpeechClient
	gigachat    *gigachat.GigaChatClient

	lgr *zap.Logger
}

func NewTelegramBot(loggger *zap.Logger, token string, pollingPeriod int, ss *salutspeech.SalutSpeechClient, gg *gigachat.GigaChatClient) (*TelegramBot, error) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: time.Duration(pollingPeriod) * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	bot := &TelegramBot{
		bot:         b,
		salutSpeech: ss,
		gigachat:    gg,
		lgr:         loggger,
	}

	b.Handle("/hello", func(c tele.Context) error {
		user := c.Sender() // кто отправил команду
		js, _ := json.Marshal(user)
		fmt.Println(string(js))
		return c.Send("Hello!")
	})

	b.Handle(tele.OnAudio, func(c tele.Context) error {
		a := c.Message().Audio
		if a == nil {
			return c.Send("Аудио не найдено")
		}

		filename := a.FileName
		if filename == "" {
			filename = "audio_" + a.FileID + ".bin"
		}

		err := ensureUploadDir("./downloads")
		if err != nil {
			c.Send("Не удалось создать папку ./downloads %s", err.Error())
		}

		path := filepath.Join("downloads", filename)

		logger.Info("Путь " + path)

		if err := c.Bot().Download(a.MediaFile(), path); err != nil {
			return c.Send(fmt.Sprintf("Не удалось скачать аудио, %s", err.Error()))
		}

		in := &model.InputAudio{
			FileName: filename,
			MIME:     a.MIME,
			IsVoice:  false,
		}

		task, err := salutSpeech.RecognizeFile(ctx, path, in)
		if err != nil {
			log.Fatal(err)
		}
		<-task.Done

		return c.Send(fmt.Sprintf(
			"Получил аудио\nНазвание: %s\nИсполнитель: %s\nMIME: %s\nДлительность: %d сек\nСохранил: %s, %s",
			a.Title,
			a.Performer,
			a.MIME,
			a.Duration,
			path,
			task.Result[0:40],
		))

	})

	b.Handle(tele.OnVoice, func(c tele.Context) error {
		a := c.Message().Voice
		if a == nil {
			return c.Send("Аудио не найдено")
		}

		filename := "voice1" + guessVoiceExt(a.MIME)
		if filename == "" {
			filename = "audio_" + a.FileID + ".bin"
		}

		err := ensureUploadDir("./downloads")
		if err != nil {
			c.Send("Не удалось создать папку ./downloads %s", err.Error())
		}

		path := filepath.Join("downloads", filename)

		logger.Info("Путь " + path)

		if err := c.Bot().Download(a.MediaFile(), path); err != nil {
			return c.Send(fmt.Sprintf("Не удалось скачать аудио, %s", err.Error()))
		}

		in := &model.InputAudio{
			FileName: filename,
			MIME:     a.MIME,
			IsVoice:  true,
		}

		task, err := salutSpeech.RecognizeFile(ctx, path, in)
		if err != nil {
			log.Fatal(err)
		}
		<-task.Done

		return c.Send(fmt.Sprintf(
			"Получил аудио \nMIME: %s\nДлительность: %d сек\nСохранил: %s, %s",
			a.MIME,
			a.Duration,
			path,
			task.Result[0:40],
		))

	})

	return bot, nil

}

func (t *TelegramBot) Start() {
	t.bot.Start()
}

func (t *TelegramBot) Stop() {
	t.bot.Stop()
}

func ensureUploadDir(basePath string) error {
	err := os.MkdirAll(basePath, 0755)
	if err != nil {
		return fmt.Errorf("не удалось создать папку загрузки: %w", err)
	}
	return nil
}

func guessVoiceExt(mime string) string {
	if strings.Contains(mime, "ogg") {
		return ".ogg"
	}
	return ".bin"
}
