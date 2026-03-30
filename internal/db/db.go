package db

import (
	"log"
	"time"

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
		Logger: logger.New(log.New(log.Writer(), "gorm ", log.LstdFlags|log.Lmicroseconds), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		}),
	})
	if err != nil {
		return err
	}

	// Auto migrate
	return DB.AutoMigrate(&models.Diary{}, &models.Goal{}, &models.GoalRecord{}, &models.SessionRevocation{})
}
