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

type GigaChatClient struct {
	client   *resty.Client
	authHost string /*TODO сделать из конфигов*/
	mainHost string /*TODO сделать из конфигов*/
	authKey  string /*TODO сделать из конфигов*/
	token    *model.Token
	mx       sync.RWMutex
	lgr      *zap.Logger
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

func (ss *GigaChatClient) setToken(token *model.Token) {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	ss.token = token
}
func (ss *GigaChatClient) GetToken() (*model.Token, error) {
	rqUID := model.NewUUID()

	response, err := ss.client.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Accept", "application/json").
		SetHeader("RqUID", rqUID).
		SetHeader("Authorization", "Basic "+ss.authKey).
		SetFormData(map[string]string{
			"scope": "SALUTE_SPEECH_PERS",
		}).
		Post(ss.authHost)

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
