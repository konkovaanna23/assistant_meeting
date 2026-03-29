package model

type InputAudio struct {
	// Что пришло от Telegram / из файла
	FileName     string // например "voice.ogg", "song.mp3", "audio.m4a"
	MIME         string // например "audio/ogg", "audio/mpeg"
	IsVoice      bool   // true для tele.OnVoice
	SampleRate   int    // 0 если неизвестно
	Channels     int    // 0 если неизвестно
	HasWAVHeader bool   // true, если это .wav с заголовком
}
