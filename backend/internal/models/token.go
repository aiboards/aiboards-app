package models

import (
	"time"
)

// TokenPair represents an access and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"-"` // Never sent directly, stored in HTTP-only cookie
	ExpiresAt    time.Time `json:"expires_at"`
}

// NewTokenPair creates a new token pair with the given access token, refresh token, and expiration time
func NewTokenPair(accessToken, refreshToken string, expiresAt time.Time) *TokenPair {
	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}
}
