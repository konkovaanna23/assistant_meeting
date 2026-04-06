package processor

import (
	"context"
	"fmt"

	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/konkovaanna23/assistant_meeting/internal/model"
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
		FileName: filename,
	}

	return p.createTask(c, job)

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

	userID := c.Sender().ID

	ext := filepath.Ext(filename)             // .mp3
	name := strings.TrimSuffix(filename, ext) // song
	filename = fmt.Sprintf("%s_%d%s", name, userID, ext)

	path := filepath.Join(dirDownloads, filename)

	if err := c.Bot().Download(a.MediaFile(), path); err != nil {
		return c.Send(fmt.Sprintf("Не удалось скачать аудио, %s", err.Error()))
	}

	handler := "OnAudio"

	job := Job{
		UserID:   userID,
		Handler:  handler,
		ChatID:   c.Chat().ID,
		FilePath: path,
		MIME:     a.MIME,
		FileName: filename,
	}

	return p.createTask(c, job)
}

func (p *ProcessorBot) GigaChatRequest(c tele.Context) error {
	p.KeepLastMessageOnly(c.Sender().ID)

	return c.Send("Жду ваш вопрос к GigaChat")
}

func (p *ProcessorBot) handlerOnText(c tele.Context) error {

	job := Job{
		UserID:  c.Sender().ID,
		Handler: "OnText",
		ChatID:  c.Chat().ID,
		Text:    c.Message().Text,
	}

	return p.createTask(c, job)
}

func (p *ProcessorBot) handlerItemGet(c tele.Context) error {

	cb := c.Callback()
	if cb == nil {
		return nil
	}

	fmt.Printf("data [%s] \n", cb.Data)
	userID := c.Sender().ID

	resultShort, result, isVoice, err := p.repo.GetAudioByID(context.Background(), userID, cb.Data)
	if err != nil {
		p.lgr.Error("ошибка получения по id", zap.String("id", cb.Data), zap.Error(err))
		return c.Respond(&tele.CallbackResponse{
			Text:      "Элемент не найден",
			ShowAlert: true,
		})
	}

	p.saveSessionRequest(userID, model.RoleUser, p.gigachat.GetTextRequest(result, isVoice))
	p.saveSessionRequest(userID, model.RoleAssistant, resultShort)

	fmt.Println(cb.Data)

	if err := c.Respond(); err != nil {
		return err
	}

	fmt.Println("Respond")

	return c.Send(resultShort)
}
