package processor

import "fmt"

type Option[T any] func(*T)

func Apply[T any](target *T, opts ...Option[T]) {
	for _, opt := range opts {
		opt(target)
	}
}

type ProcessorBotConfig struct {
	SizeChannel      int
	CountWorkers     int
	PollingPeriodBot int
	UploadDir        string
}

type ProcessorBotOption = Option[ProcessorBotConfig]

func WithSizeChannel(n int) ProcessorBotOption {
	return func(c *ProcessorBotConfig) {
		c.SizeChannel = n
	}
}

func WithCountWorkers(n int) ProcessorBotOption {
	return func(c *ProcessorBotConfig) {
		c.CountWorkers = n
	}
}

func WithPollingPeriodBot(seconds int) ProcessorBotOption {
	return func(c *ProcessorBotConfig) {
		c.PollingPeriodBot = seconds
	}
}

func WithUploadDir(dir string) ProcessorBotOption {
	return func(c *ProcessorBotConfig) {
		c.UploadDir = dir
	}
}

func (c *ProcessorBotConfig) Validate() error {
	if c.SizeChannel <= 0 {
		return fmt.Errorf("sizeChannel не может быть меньше 0")
	}
	if c.CountWorkers <= 0 {
		return fmt.Errorf("countWorkers не может быть меньше 0")
	}
	if c.PollingPeriodBot <= 0 {
		return fmt.Errorf("pollingPeriodBot не может быть меньше 0")
	}
	if c.UploadDir == "" {
		return fmt.Errorf("не указана директория для хранения  временных файлов")
	}

	return nil
}
