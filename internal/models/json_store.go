package models

import "time"

type JSONStoreItem struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"size:64;not null;uniqueIndex" json:"key"`
	Content   string    `gorm:"type:text;not null" json:"-"`
	SizeBytes int64     `gorm:"not null" json:"size_bytes"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `gorm:"index" json:"updated_at"`
}
