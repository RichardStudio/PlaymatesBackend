package models

import (
	"time"
)

type RefreshToken struct {
	ID          int
	UserID      int
	Username    string
	HashedToken string
	Revoked     bool
	ExpiresAt   time.Time
	Fingerprint string
	CreatedAt   time.Time
}
