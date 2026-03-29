package models

import (
	"time"
)

type Diary struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	CreateDate string    `gorm:"type:varchar(10);not null;index" json:"create_date"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type User struct {
	Username string `json:"username"`
	Password string `json:"-"` // Never expose password
}
