package model

import (
	"fmt"
	"strings"
)

type InputAudio struct {
	// Что пришло от Telegram / из файла
	AudioID      string
	FileName     string // например "voice.ogg", "song.mp3", "audio.m4a"
	MIME         string // например "audio/ogg", "audio/mpeg"
	IsVoice      bool   // true для tele.OnVoice
	SampleRate   int    // 0 если неизвестно
	Channels     int    // 0 если неизвестно
	HasWAVHeader bool   // true, если это .wav с заголовком
}

type AudioShort struct {
	ID   string
	Path string
}

type Task struct {
	AudioID        string
	PathFile       string
	FileID         string
	TaskID         string
	Status         string
	Done           chan struct{}
	Result         string
	ErrorMessage   string
	ResponseFileID string
	FunsSaveStatus func(id, taskID, fileID, status string) error
}

func FormatAudioList(audioList []*AudioShort) string {
	if len(audioList) == 0 {
		return "🎙 У вас пока нет сохранённых аудиозаписей."
	}

	var lines []string

	lines = append(lines, "📁 Список ваших аудиозаписей:")
	lines = append(lines, "")
	lines = append(lines, " №  | ID                          | Файл")
	lines = append(lines, "────┼─────────────────────────────┼──────────────────────")

	for i, audio := range audioList {

		fileName := audio.Path
		if lastSlash := strings.LastIndex(fileName, "/"); lastSlash != -1 {
			fileName = fileName[lastSlash+1:]
		}

		if len(fileName) > 20 {
			fileName = fileName[:18] + ".."
		}

		line := fmt.Sprintf(" %d  | %s | %s",
			i+1,
			audio.ID,
			fileName,
		)
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Всего: %d файл(а/ов)", len(audioList)))

	return strings.Join(lines, "\n")
}
