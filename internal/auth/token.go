package auth

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/konkovaanna23/assistant_meeting/internal/model"
)

type FetchTokenFunc func() (*model.Token, error)

type TokenManager struct {
	mx        sync.RWMutex
	refreshMx sync.Mutex

	token *model.Token
	fetch FetchTokenFunc
}

func NewTokenManager(fetch FetchTokenFunc) *TokenManager {
	return &TokenManager{
		fetch: fetch,
	}
}

func (tm *TokenManager) SetToken(token *model.Token) {
	tm.mx.Lock()
	defer tm.mx.Unlock()
	tm.token = token
}

func (tm *TokenManager) CurrentToken() string {
	tm.mx.RLock()
	defer tm.mx.RUnlock()

	if tm.token == nil {
		return ""
	}

	return tm.token.Token
}

func (tm *TokenManager) RefreshToken() error {
	tm.refreshMx.Lock()
	defer tm.refreshMx.Unlock()

	if tm.CurrentToken() != "" {
		return nil
	}

	token, err := tm.fetch()
	if err != nil {
		return err
	}

	tm.SetToken(token)
	return nil
}

func (tm *TokenManager) ForceRefreshToken() error {
	tm.refreshMx.Lock()
	defer tm.refreshMx.Unlock()

	token, err := tm.fetch()
	if err != nil {
		return err
	}

	tm.SetToken(token)
	return nil
}

func (tm *TokenManager) DoWithRetry(req *resty.Request, method, url string) (*resty.Response, error) {

	if tm.CurrentToken() == "" {
		if err := tm.RefreshToken(); err != nil {
			return nil, fmt.Errorf("не удалось получить токен: %w", err)
		}
	}

	req.SetHeader("Authorization", "Bearer "+tm.CurrentToken())

	resp, err := req.Execute(method, url)
	if err == nil && resp.StatusCode() != http.StatusUnauthorized {
		return resp, nil
	}

	if errRefresh := tm.ForceRefreshToken(); errRefresh != nil {
		return nil, fmt.Errorf("не удалось обновить токен: %w", errRefresh)
	}

	req.SetHeader("Authorization", "Bearer "+tm.CurrentToken())

	resp, err = req.Execute(method, url)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
