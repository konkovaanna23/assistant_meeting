package model

import (
	"context"
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
	FunsSaveStatus func(ctx context.Context, id, taskID, fileID, status string) error
	CtxTask        context.Context
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

type WordAlignment struct {
	Word  string `json:"word"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type Result struct {
	Text           string          `json:"text"`
	NormalizedText string          `json:"normalized_text"`
	Start          string          `json:"start"`
	End            string          `json:"end"`
	WordAlignments []WordAlignment `json:"word_alignments"`
}

type Chunk struct {
	Results []Result `json:"results"`
}

type CombinedResult struct {
	NormalizedText string          `json:"normalized_text"`
	WordAlignments []WordAlignment `json:"word_alignments"`
}

// MergeNormalizedTextAndUniqueWords собирает normalized_text в одну строку
// и возвращает уникальные word_alignments по полю word.
// Уникальность определяется по слову без учета регистра.
func MergeNormalizedTextAndUniqueWords(chunks []Chunk) *CombinedResult {
	var textParts []string
	uniqueWords := make([]WordAlignment, 0)
	seen := make(map[string]struct{})

	for _, chunk := range chunks {
		for _, res := range chunk.Results {
			normalizedText := strings.TrimSpace(res.NormalizedText)
			if normalizedText != "" {
				textParts = append(textParts, normalizedText)
			}

			for _, wa := range res.WordAlignments {
				word := strings.TrimSpace(wa.Word)
				if word == "" {
					continue
				}

				key := strings.ToLower(word)
				if _, exists := seen[key]; exists {
					continue
				}

				seen[key] = struct{}{}
				uniqueWords = append(uniqueWords, wa)
			}
		}
	}

	return &CombinedResult{
		NormalizedText: strings.Join(textParts, " "),
		WordAlignments: uniqueWords,
	}
}

func BuildCombinedJSON(chunks []Chunk) ([]byte, error) {
	result := MergeNormalizedTextAndUniqueWords(chunks)
	return json.MarshalIndent(result, "", "  ")
}

/*

	data, err := os.ReadFile("result.txt")
	if err != nil {
		panic(err)
	}

	if err := json.Unmarshal(data, &chunks); err != nil {
		panic(err)
	}

	out, err := BuildCombinedJSON(chunks)
	if err != nil {
		panic(err)
	}
*/
