package model

type WordAlignment struct {
	Word  string `json:"word"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type Result struct {
	Text           string           `json:"text"`
	NormalizedText string           `json:"normalized_text"`
	Start          string           `json:"start"`
	End            string           `json:"end"`
	WordAlignments []*WordAlignment `json:"word_alignments"`
}

type Chunk struct {
	Results []Result `json:"results"`
}

type CombinedResult struct {
	NormalizedText string           `json:"normalized_text"`
	WordAlignments []*WordAlignment `json:"word_alignments"`
}
