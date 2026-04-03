package processor

import (
	"encoding/json"
	"strings"

	"github.com/konkovaanna23/assistant_meeting/internal/model"
)

// MergeNormalizedTextAndUniqueWords собирает normalized_text в одну строку
// и возвращает уникальные word_alignments по полю word.
// Уникальность определяется по слову без учета регистра.
func MergeNormalizedTextAndUniqueWords(chunks []*model.Chunk) *model.CombinedResult {
	var textParts []string
	uniqueWords := make([]*model.WordAlignment, 0)
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

	return &model.CombinedResult{
		NormalizedText: strings.Join(textParts, " "),
		WordAlignments: uniqueWords,
	}
}

func BuildCombinedJSON(text string) (*model.CombinedResult, error) {
	var chunks []*model.Chunk

	if err := json.Unmarshal([]byte(text), &chunks); err != nil {
		return nil, err
	}

	result := MergeNormalizedTextAndUniqueWords(chunks)
	return result, nil
}
