package salutspeech

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/konkovaanna23/assistant_meeting/internal/model"
)

type AudioSpec struct {
	Encoding    string // PCM_S16LE, OPUS, MP3, FLAC, ALAW, MULAW
	ContentType string // что ставить в HTTP Content-Type
	SampleRate  int    // что передавать в config, если нужно
	Channels    int
}

func DetectAudioSpec(in *model.InputAudio) (*AudioSpec, error) {
	ext := strings.ToLower(filepath.Ext(in.FileName))
	mime := strings.ToLower(in.MIME)

	if in.IsVoice || ext == ".ogg" || ext == ".opus" ||
		strings.Contains(mime, "ogg") || strings.Contains(mime, "opus") {
		return &AudioSpec{
			Encoding:    "OPUS",
			ContentType: "audio/ogg;codecs=opus",
			SampleRate:  0,
			Channels:    1,
		}, nil
	}

	if ext == ".mp3" || strings.Contains(mime, "mpeg") || strings.Contains(mime, "mp3") {
		ch := in.Channels
		if ch == 0 {
			ch = 1
		}
		if ch > 2 {
			return nil, errors.New("mp3 поддерживается максимум с 2 каналами; лучше привести к mono или stereo")
		}
		return &AudioSpec{
			Encoding:    "MP3",
			ContentType: "audio/mpeg",
			SampleRate:  0,
			Channels:    ch,
		}, nil
	}

	if ext == ".flac" || strings.Contains(mime, "flac") {
		ch := in.Channels
		if ch == 0 {
			ch = 1
		}
		if ch > 8 {
			return nil, errors.New("flac поддерживается максимум с 8 каналами")
		}
		return &AudioSpec{
			Encoding:    "FLAC",
			ContentType: "audio/flac",
			SampleRate:  0,
			Channels:    ch,
		}, nil
	}

	// PCM_S16LE: сюда относим .wav/.pcm, но только если это реально PCM 16-bit LE.
	if ext == ".wav" || ext == ".pcm" || strings.Contains(mime, "wav") || strings.Contains(mime, "pcm") {
		rate := in.SampleRate
		if rate == 0 {
			rate = 16000
		}
		ch := in.Channels
		if ch == 0 {
			ch = 1
		}

		if ch > 8 {
			return nil, errors.New("pcm_s16le поддерживается максимум с 8 каналами")
		}

		return &AudioSpec{
			Encoding:    "PCM_S16LE",
			ContentType: fmt.Sprintf("audio/x-pcm;bit=16;rate=%d", rate),
			SampleRate:  rate,
			Channels:    ch,
		}, nil
	}

	if ext == ".alaw" || strings.Contains(mime, "pcma") || strings.Contains(mime, "alaw") {
		rate := in.SampleRate
		if rate == 0 {
			rate = 8000
		}
		if rate != 8000 {
			return nil, errors.New("alaw поддерживается только с частотой 8000 Гц")
		}
		if in.Channels > 1 {
			return nil, errors.New("alaw поддерживается только в mono")
		}
		return &AudioSpec{
			Encoding:    "ALAW",
			ContentType: "audio/pcma;rate=8000",
			SampleRate:  8000,
			Channels:    1,
		}, nil
	}

	if ext == ".mulaw" || ext == ".ulaw" || strings.Contains(mime, "pcmu") || strings.Contains(mime, "mulaw") {
		rate := in.SampleRate
		if rate == 0 {
			rate = 8000
		}
		if rate != 8000 {
			return nil, errors.New("mulaw поддерживается только с частотой 8000 Гц")
		}
		if in.Channels > 1 {
			return nil, errors.New("mulaw поддерживается только в mono")
		}
		return &AudioSpec{
			Encoding:    "MULAW",
			ContentType: "audio/pcmu;rate=8000",
			SampleRate:  8000,
			Channels:    1,
		}, nil
	}

	return nil, errors.New("формат не входит в поддерживаемый список; лучше перекодировать в mp3 или wav pcm_s16le")
}
