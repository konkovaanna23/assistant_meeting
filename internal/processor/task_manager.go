package processor

import (
	"context"
	"errors"
)

func (p *ProcessorBot) process(ctx context.Context, job *Job) (string, error) {
	switch job.Handler {
	case "/start":
		err := p.repo.CreateUser(ctx, job.UserID, job.ChatID, job.Username)
		if err != nil {
			return "", err
		}

	default:
		return "", errors.New("неизвестная команда")
	}
	return "", nil
}
