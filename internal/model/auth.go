package model

import "github.com/google/uuid"

type Token struct {
	Token       string `json:"access_token"`
	ExpiresDate int64  `json:"expires_at"`
}

func NewUUID() string {
	return uuid.New().String()
}
