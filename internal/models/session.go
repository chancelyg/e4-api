package models

import "time"

type SessionRevocation struct {
	SessionID string    `gorm:"primaryKey;size:64" json:"session_id"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
