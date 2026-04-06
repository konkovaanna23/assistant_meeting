package processor

import (
	"time"

	"github.com/konkovaanna23/assistant_meeting/internal/model"
	"go.uber.org/zap"
)

func (p *ProcessorBot) saveSessionRequest(userID int64, role, message string) {
	p.mx.Lock()
	defer p.mx.Unlock()

	if p.userHistory == nil {
		p.userHistory = make(map[int64][]*model.Message)
	}

	messages := p.userHistory[userID]
	messages = append(messages, &model.Message{
		Role:    role,
		Content: message,
	})
	p.userHistory[userID] = messages
}

func (p *ProcessorBot) StartCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			p.cleanupSessions()
		}
	}()
}

func (p *ProcessorBot) cleanupSessions() {
	p.mx.Lock()
	defer p.mx.Unlock()

	const maxMessages = 10

	p.lgr.Info("Начинаем очистку сессий...", zap.Int("текущих пользователей", len(p.userHistory)))

	for userID, messages := range p.userHistory {
		if len(messages) > maxMessages {
			p.userHistory[userID] = messages[len(messages)-maxMessages:]
			p.lgr.Info("История пользователя обрезана", zap.Int64("пользователь", userID), zap.Int("количество сообщение", maxMessages))
		}
	}

	p.lgr.Info("Очистка завершена")
}

func (p *ProcessorBot) GetSession(userID int64) []*model.Message {
	p.mx.RLock()
	defer p.mx.RUnlock()

	messages, exists := p.userHistory[userID]
	if !exists || len(messages) == 0 {
		return nil
	}

	messagesCopy := make([]*model.Message, len(messages))
	copy(messagesCopy, messages)
	return messagesCopy
}

func (p *ProcessorBot) KeepLastMessageOnly(userID int64) {
	p.mx.Lock()
	defer p.mx.Unlock()

	messages, exists := p.userHistory[userID]
	if !exists || len(messages) == 0 {
		return
	}

	last := messages[len(messages)-1]
	p.userHistory[userID] = []*model.Message{last}
}
