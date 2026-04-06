package gigachat

import (
	"context"
	"encoding/json"
	"errors"
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
	Model       string           `json:"model"`
	Messages    []*model.Message `json:"messages"`
	Temperature float64          `json:"temperature"`
	MaxTokens   int              `json:"max_tokens"`
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
	err := gigaChat.refreshToken()
	if err != nil {
		return nil, fmt.Errorf("ошибка обновления токена: %s", err.Error())
	}

	return gigaChat, nil
}

func (gg *GigaChatClient) refreshToken() error {
	responseToken, err := gg.GetToken()
	if err != nil {
		return err
	}
	gg.setToken(responseToken)

	return nil
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

func (gg *GigaChatClient) DoWithRetry(req *resty.Request, method, url string) (*resty.Response, error) {
	resp, err := req.Execute(method, url)

	if err == nil && resp.StatusCode() != 401 {
		return resp, err
	}

	if errRefresh := gg.refreshToken(); errRefresh != nil {
		return nil, fmt.Errorf("не удалось обновить токен: %w", errRefresh)
	}

	req.SetHeader("Authorization", "Bearer "+gg.GetCurrentToken())

	return req.Execute(method, url)
}

func (gg *GigaChatClient) GetCurrentToken() string {
	gg.mx.RLock()
	defer gg.mx.RUnlock()
	return gg.token.Token
}

func (gg *GigaChatClient) GetTextRequest(text string, isVoice bool) string {
	promptRequest := ""

	if isVoice {
		promptRequest = promptVoice + text
	} else {
		promptRequest = promptAudio + text
	}

	return promptRequest
}

func (gg *GigaChatClient) GetBriefExtract(text string, isVoice bool) (string, error) {
	if text == "" {
		return "", errors.New("пустой запрос на получение краткой выжимки")
	}

	promptRequest := gg.GetTextRequest(text, isVoice)

	reqBody := &GigaChatRequest{
		Model: "GigaChat",
		Messages: []*model.Message{
			{Role: "user", Content: promptRequest},
		},
		Temperature: 0.5,
		MaxTokens:   1024,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	fmt.Println("Body ", string(body))
	response, err := gg.DoWithRetry(gg.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+gg.GetCurrentToken()).
		SetBody(body),
		"POST",
		gg.mainHost+endpointCompletions,
	)

	if err != nil {
		return "", err
	}

	if response.StatusCode() != http.StatusOK {
		gg.lgr.Error("ошибка запроса", zap.String("response", response.String()))
		return "", getErr(response.StatusCode())
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

func getErr(statusCode int) error {

	switch statusCode {
	case 400:
		return errors.New("некорректный формат запроса")
	case 401:
		return errors.New("ошибка аторизации")
	case 404:
		return errors.New("указан неверный идентификатор модели.")
	case 422:
		return errors.New("ошибка валидации параметров запроса, проверьте названия полей и значения параметров.")
	case 429:
		return errors.New("слишком много запросов в единицу времени.")
	default:
		return errors.New("Internal Server Error")
	}
}

func (gg *GigaChatClient) Chat(ctx context.Context, msgs []*model.Message) (string, error) {
	if len(msgs) == 0 {
		return "", errors.New("нет сообщений")
	}

	reqBody := &GigaChatRequest{
		Model:       "GigaChat",
		Messages:    msgs,
		Temperature: 0.5,
		MaxTokens:   1024,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	fmt.Println("Body ", string(body))

	gg.client.SetDebug(true)

	response, err := gg.DoWithRetry(gg.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+gg.GetCurrentToken()).
		SetBody(body),
		"POST",
		gg.mainHost+endpointCompletions,
	)

	if err != nil {
		return "", err
	}

	if response.StatusCode() != http.StatusOK {
		return "", getErr(response.StatusCode())
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
