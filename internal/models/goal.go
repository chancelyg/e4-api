package models

import "time"

const (
	GoalTypeCheckbox  = "checkbox"
	GoalTypeQuantity  = "quantity"
	GoalTypeFrequency = "frequency"
)

type Goal struct {
	ID           uint         `gorm:"primaryKey" json:"id"`
	Name         string       `gorm:"size:120;not null" json:"name"`
	GoalType     string       `gorm:"size:24;not null;index" json:"goal_type"`
	Unit         string       `gorm:"size:20" json:"unit"`
	AnnualTarget *float64     `json:"annual_target"`
	WeeklyTarget *int         `json:"weekly_target"`
	IsActive     bool         `gorm:"not null;default:true;index" json:"is_active"`
	SortOrder    int          `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	Records      []GoalRecord `json:"-"`
}

type GoalRecord struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	GoalID      uint      `gorm:"not null;index:idx_goal_record_date,unique" json:"goal_id"`
	RecordDate  string    `gorm:"type:varchar(10);not null;index:idx_goal_record_date,unique;index" json:"record_date"`
	IsCompleted bool      `gorm:"not null;default:true" json:"is_completed"`
	Quantity    *float64  `json:"quantity"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
