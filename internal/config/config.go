// Package config - пакет для работы с конфигурацией.
package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

const (
	defaultCountWorkers   = 10
	defaultSizeChannel    = 100
	defaultPeriodPolling  = 2
	defaultPollingBot     = 10
	defaultContextTimeout = 1
)

// Config Конфигурация приложения.
type Config struct {
	TelegramToken  string   `json:"token_telegram"`
	CountWorkers   int      `json:"count_workers"`
	SizeChannel    int      `json:"optimal_chan_size"`
	PeriodPolling  int      `json:"period_polling"`
	SalutSpeech    *Setting `json:"salut_speech"`
	GigaChat       *Setting `json:"giga_chat"`
	DSN            string   `json:"DSN"`
	PollingBot     int      `json:"polling_bot"`
	ContextTimeout int      `json:"context_timeout"`
}

type Setting struct {
	Token string `json:"token"`
	Auth  string `json:"auth_host"`
	Main  string `json:"request_host"`
}

func GetConfig() (*Config, error) {

	var configPath string

	fileConfigFlag := flag.String("с", "config.json", "Файл конфигурации")

	flag.CommandLine.Visit(func(f *flag.Flag) {
		if f.Name == "c" {
			configPath = *fileConfigFlag
		}
	})

	configText, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}

	err = json.Unmarshal(configText, &cfg)
	if err != nil {
		return nil, fmt.Errorf("некорректный формат файла:%s", err.Error())
	}
	if cfg.CountWorkers == 0 {
		cfg.CountWorkers = defaultCountWorkers
	}
	if cfg.SizeChannel == 0 {
		cfg.SizeChannel = defaultSizeChannel
	}

	if cfg.PeriodPolling == 0 {
		cfg.PeriodPolling = defaultPeriodPolling
	}

	if cfg.PollingBot == 0 {
		cfg.PollingBot = defaultPollingBot
	}

	if cfg.ContextTimeout == 0 {
		cfg.ContextTimeout = defaultContextTimeout
	}

	if cfg.TelegramToken == "" {
		return nil, errors.New("не задан токен для telegram-bot")
	}

	if cfg.SalutSpeech == nil {
		return nil, errors.New("не задан конфигурация для SalutSpeech")
	}

	if cfg.GigaChat == nil {
		return nil, errors.New("не задан конфигурация для GigaChat")
	}

	if cfg.DSN == "" {
		return nil, errors.New("не задан конфигурация для подключения к БД")
	}

	return cfg, nil
}
