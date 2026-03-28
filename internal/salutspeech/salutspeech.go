package salutspeech

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
)

const (
	endpointsUpload    = "/data:upload"
	endpointsRecognize = "/speech:async_recognize"
	endpointsGetStatus = "/task:get"
	endpointsGetData   = "/data:download"
)

type SalutSpeechClient struct {
	client   *resty.Client
	authHost string /*TODO сделать из конфигов*/
	mainHost string /*TODO сделать из конфигов*/
	authKey  string /*TODO сделать из конфигов*/
	token    *ResponseToken
	mx       sync.RWMutex
}

type ResponseToken struct {
	Token       string    `json:"access_token"`
	ExpiresDate time.Time `json:"expires_at"`
}

type ResultID struct {
	FileID string `json:"request_file_id"`
}

type ResponseUploadFile struct {
	Status int       `json:"status"`
	Result *ResultID `json:"result"`
}

type RequestOptions struct {
	Model         string `json:"model"`
	Encoding      string `json:"encoding"`
	SampleRate    int    `json:"sample_rate"`
	ChannelsCount int    `json:"channels_count"`
	Language      string `json:"language"`
}

type RequestRecognize struct {
	Options *RequestOptions `json:"options"`
	FileID  string          `json:"request_file_id"`
}

type StatusTask struct {
	ID        string `json:"id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	Status    string `json:"status"`
}

type ResponseRecognize struct {
	Status int         `json:"status"`
	Result *StatusTask `json:"result"`
}

func NewSalutSpeechClient(authHost, mainHost, authKey string) *SalutSpeechClient {
	return &SalutSpeechClient{
		client:   resty.New(),
		authHost: authHost,
		authKey:  authKey,
	}
}

func (ss *SalutSpeechClient) GetToken() (*ResponseToken, error) {
	rqUID := NewUUID()

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

	responseToken := &ResponseToken{}

	err = json.Unmarshal(response.Body(), &responseToken)
	if err != nil {
		return nil, fmt.Errorf("некорректный формат ответа: %s, body: %s", err, response.String())
	}
	return responseToken, nil
}

func NewUUID() string {
	return uuid.New().String()
}

func (ss *SalutSpeechClient) UploadFile(audioFilePath string) (string, error) {

	response, err := ss.client.R().
		SetHeader("Content-Type", "audio/pcmu;rate=16000").
		SetFile("audio", audioFilePath).
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+ss.GetCurrentToken()).
		Post(ss.mainHost + endpointsUpload)

	if err != nil {
		return "", err
	}

	if response.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("%s", response.Body())
	}

	responseUpload := &ResponseUploadFile{}

	err = json.Unmarshal(response.Body(), &responseUpload)
	if err != nil {
		return "", fmt.Errorf("Некорректный формат ответа: %s, body: %s", err.Error(), response.String())
	}
	/*TODO добавить обработку всех статусов*/
	if responseUpload.Status == 200 {
		return responseUpload.Result.FileID, nil
	} else {
		return "", fmt.Errorf("некорректный ответ body: %s", response.String())
	}

}

func (ss *SalutSpeechClient) GetCurrentToken() string {
	ss.mx.RLock()
	defer ss.mx.RUnlock()
	return ss.token.Token
}

func (ss *SalutSpeechClient) CreateTaskRecognize(fileID string) (*StatusTask, error) {

	request := &RequestRecognize{
		Options: &RequestOptions{Model: "general",
			Encoding:      "PCM_S16LE",
			SampleRate:    16000,
			Language:      "ru-RU",
			ChannelsCount: 1,
		},
		FileID: fileID,
	}

	requestJson, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("некорректный формат запроса %s", err.Error())
	}

	response, err := ss.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+ss.GetCurrentToken()).
		SetBody(requestJson).
		Post(ss.mainHost + endpointsRecognize)

	if err != nil {
		return nil, err
	}

	if response.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%s", response.Body())
	}

	responseRecognize := &ResponseRecognize{}

	err = json.Unmarshal(response.Body(), &responseRecognize)
	if err != nil {
		return nil, fmt.Errorf("Некорректный формат ответа: %s, body: %s", err, response.String())
	}
	/*TODO добавить обработку всех статусов*/
	if responseRecognize.Status == 200 {
		return responseRecognize.Result, nil
	} else {
		return nil, fmt.Errorf("Некорректный ответ: %s", response.String())
	}

}

func (ss *SalutSpeechClient) GetStatusTask(taskID string) (string, error) {

	response, err := ss.client.R().
		SetHeader("Accept", "application/octet-stream").
		SetHeader("Authorization", "Bearer "+ss.GetCurrentToken()).
		SetQueryParams(map[string]string{
			"id": taskID,
		}).
		Get(ss.mainHost + endpointsGetStatus)

	if err != nil {
		return "", err
	}

	if response.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("%s", response.Body())
	}

	responseStatus := &StatusTask{}

	err = json.Unmarshal(response.Body(), &response)
	if err != nil {
		return "", fmt.Errorf("Некорректный формат ответа: %s, body: %s", err, response.String())
	}
	/*TODO добавить обработку всех статусов*/
	return responseStatus.Status, nil

}

func (ss *SalutSpeechClient) GetData(fileID string) (string, error) {

	response, err := ss.client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+ss.GetCurrentToken()).
		SetQueryParams(map[string]string{
			"response_file_id": fileID,
		}).
		Get(ss.mainHost + endpointsGetData)

	if err != nil {
		return "", err
	}

	if response.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("%s", response.Body())
	}

	/*TODO добавить обработку всех статусов*/
	return string(response.Body()), nil

}
