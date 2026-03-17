package repositories

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"myguy/internal/models"
)

func setupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the models
	err = db.AutoMigrate(
		&models.User{},
		&models.Task{},
		&models.Application{},
		&models.Review{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}
