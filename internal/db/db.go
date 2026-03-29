package db

import (
	"e4-api/internal/config"
	"e4-api/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init() error {
	var err error

	DB, err = gorm.Open(sqlite.Open(config.Cfg.Database.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}

	// Auto migrate
	return DB.AutoMigrate(&models.Diary{}, &models.Goal{}, &models.GoalRecord{}, &models.SessionRevocation{})
}
