package processor

import (
	"fmt"

	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

const (
	dirDownloads = "downloads"
)

func (p *ProcessorBot) createTask(c tele.Context, job Job) error {

	select {
	case p.jobs <- job:
		p.lgr.Info("Задача по команде создана", zap.String("command", job.Handler))
	default:
		return c.Send("Очередь переполнена, попробуй чуть позже")
	}

	return nil
}

func (p *ProcessorBot) SaveUser(c tele.Context) error {

	handler := "/start"

	job := Job{
		UserID:   c.Sender().ID,
		Username: c.Sender().Username,
		Handler:  handler,
		ChatID:   c.Chat().ID,
	}

	return p.createTask(c, job)
}

func (p *ProcessorBot) GetListAudio(c tele.Context) error {
	handler := "/list"

	job := Job{
		UserID:  c.Sender().ID,
		Handler: handler,
		ChatID:  c.Chat().ID,
	}

	return p.createTask(c, job)

}

func (p *ProcessorBot) GetTextAudio(c tele.Context) error {
	handler := "/get"

	args := c.Args()
	if len(args) < 1 {
		return c.Send("Использование: /get <id>")
	}

	id := args[0]

	job := Job{
		UserID:  c.Sender().ID,
		Handler: handler,
		ChatID:  c.Chat().ID,
		IdAudio: id,
	}

	return p.createTask(c, job)

}

func (p *ProcessorBot) FindAudio(c tele.Context) error {
	handler := "/find"

	args := c.Args()
	if len(args) < 1 {
		return c.Send("Использование: /find <word>")
	}

	word := args[0]

	job := Job{
		UserID:  c.Sender().ID,
		Handler: handler,
		ChatID:  c.Chat().ID,
		KeyWord: word,
	}

	return p.createTask(c, job)

}

func (p *ProcessorBot) hanlerOnVoice(c tele.Context) error {
	v := c.Message().Voice
	if v == nil {
		return c.Send("Голосовое сообщение не найдено")
	}

	filename := GenerateVoiceFilename(c.Sender().Username)

	path := filepath.Join(dirDownloads, filename)

	if err := c.Bot().Download(v.MediaFile(), path); err != nil {
		return c.Send(fmt.Sprintf("Не удалось скачать аудио, %s", err.Error()))
	}

	handler := "OnVoice"

	job := Job{
		UserID:   c.Sender().ID,
		Handler:  handler,
		ChatID:   c.Chat().ID,
		FilePath: path,
	}

	return p.createTask(c, job)

	/*in := &model.InputAudio{
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
	))*/

}

func ensureUploadDir(basePath string) error {
	err := os.MkdirAll(basePath, 0755)
	if err != nil {
		return fmt.Errorf("не удалось создать папку загрузки: %w", err)
	}
	return nil
}

func GenerateVoiceFilename(username string) string {
	cleanUsername := strings.ToLower(strings.TrimSpace(username))
	if cleanUsername == "" {
		cleanUsername = "unknown"
	}

	for _, r := range []string{"@", ".", " ", "-", "#"} {
		cleanUsername = strings.ReplaceAll(cleanUsername, r, "_")
	}

	timestamp := time.Now().Format("20060102150405")

	return fmt.Sprintf("voice_%s_%s.ogg", cleanUsername, timestamp)
}

func (p *ProcessorBot) handlerOnAudio(c tele.Context) error {
	a := c.Message().Audio
	if a == nil {
		return c.Send("Аудио не найдено")
	}

	filename := a.FileName
	if filename == "" {
		c.Send("Не удалось определить имя файла")
	}

	path := filepath.Join(dirDownloads, filename)

	if err := c.Bot().Download(a.MediaFile(), path); err != nil {
		return c.Send(fmt.Sprintf("Не удалось скачать аудио, %s", err.Error()))
	}

	handler := "OnAudio"

	job := Job{
		UserID:   c.Sender().ID,
		Handler:  handler,
		ChatID:   c.Chat().ID,
		FilePath: path,
	}

	return p.createTask(c, job)
	/*	in := &model.InputAudio{
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
		))*/

}
