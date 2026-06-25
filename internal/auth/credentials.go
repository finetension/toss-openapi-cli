package auth

import (
	"encoding/json"
	"time"
)

const TokenRefreshBuffer = 60 * time.Second

type Credentials struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type CachedToken struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

func EncodeCredentials(credentials Credentials) (string, error) {
	b, err := json.Marshal(credentials)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DecodeCredentials(value string) (Credentials, error) {
	var credentials Credentials
	err := json.Unmarshal([]byte(value), &credentials)
	return credentials, err
}

func EncodeCachedToken(token CachedToken) (string, error) {
	b, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DecodeCachedToken(value string) (CachedToken, error) {
	var token CachedToken
	err := json.Unmarshal([]byte(value), &token)
	return token, err
}

func (t CachedToken) Valid(now time.Time) bool {
	if t.AccessToken == "" || t.ExpiresAt.IsZero() {
		return false
	}
	return t.ExpiresAt.After(now.Add(TokenRefreshBuffer))
}
