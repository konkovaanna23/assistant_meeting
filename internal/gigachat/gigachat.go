package gigachat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/konkovaanna23/assistant_meeting/internal/config"
	"github.com/konkovaanna23/assistant_meeting/internal/model"
	"go.uber.org/zap"
)

const (
	endpointCompletions = "/chat/completions"
)

var promptAudio string = `Сделай краткую выжимку из расшифровки совещания: суть обсуждения, ключевые решения, договоренности, задачи, ответственные и сроки. Коротко, структурированно, без воды, ничего не додумывай. Из этого текста `
var promptVoice string = `Сделай краткую выжимку из расшифровки голосового сообщения: суть, важные факты, просьбы/поручения, следующие действия. Коротко, без воды, ничего не додумывай. `

type GigaChatClient struct {
	client   *resty.Client
	authHost string /*TODO сделать из конфигов*/
	mainHost string /*TODO сделать из конфигов*/
	authKey  string /*TODO сделать из конфигов*/
	token    *model.Token
	mx       sync.RWMutex
	lgr      *zap.Logger
}

type GigaChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GigaChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewGigaChatClient(ctx context.Context, setting *config.Setting, logger *zap.Logger) (*GigaChatClient, error) {
	gigaChat := &GigaChatClient{
		client:   resty.New(),
		authHost: setting.Auth,
		authKey:  setting.Token,
		mainHost: setting.Main,
		lgr:      logger,
	}
	responseToken, err := gigaChat.GetToken()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения токена: %s", err.Error())
	}
	gigaChat.setToken(responseToken)

	return gigaChat, nil
}

func (gg *GigaChatClient) setToken(token *model.Token) {
	gg.mx.Lock()
	defer gg.mx.Unlock()
	gg.token = token
}
func (gg *GigaChatClient) GetToken() (*model.Token, error) {
	rqUID := model.NewUUID()

	response, err := gg.client.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Accept", "application/json").
		SetHeader("RqUID", rqUID).
		SetHeader("Authorization", "Basic "+gg.authKey).
		SetFormData(map[string]string{
			"scope": "GIGACHAT_API_PERS",
		}).
		Post(gg.authHost)

	if err != nil {
		return nil, err
	}

	if response.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%s", response.Body())
	}

	responseToken := &model.Token{}

	err = json.Unmarshal(response.Body(), &responseToken)
	if err != nil {
		return nil, fmt.Errorf("некорректный формат ответа: %s, body: %s", err, response.String())
	}
	return responseToken, nil
}

func (gg *GigaChatClient) GetCurrentToken() string {
	gg.mx.RLock()
	defer gg.mx.RUnlock()
	return gg.token.Token
}

func (gg *GigaChatClient) GetBriefExtract(text string, isVoice bool) (string, error) {
	promptRequest := ""
	if isVoice {
		promptRequest = promptVoice + text
	} else {
		promptRequest = promptAudio + text
	}

	reqBody := &GigaChatRequest{
		Model: "GigaChat",
		Messages: []Message{
			{Role: "user", Content: promptRequest},
		},
		Temperature: 0.5,
		MaxTokens:   1024,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	gg.client.SetDebug(true)

	response, err := gg.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+gg.GetCurrentToken()).
		SetBody(body).
		Post(gg.mainHost + endpointCompletions)

	if err != nil {
		return "", err
	}

	if response.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("%s", response.Body())
	}

	responseGG := &GigaChatResponse{}

	err = json.Unmarshal(response.Body(), &responseGG)
	if err != nil {
		return "", fmt.Errorf("некорректный формат ответа: %s, body: %s", err, response.String())
	}
	if len(responseGG.Choices) > 0 {
		return responseGG.Choices[0].Message.Content, nil
	}

	return "Нет ответа", nil
}
