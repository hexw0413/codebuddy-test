package database

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"csgo-trader/internal/models"
)

func Initialize(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto migrate the schema
	err = db.AutoMigrate(
		&models.User{},
		&models.Item{},
		&models.Price{},
		&models.Trade{},
		&models.Strategy{},
		&models.Inventory{},
		&models.MarketTrend{},
	)
	if err != nil {
		return nil, err
	}

	log.Println("Database initialized successfully")
	return db, nil
}

func GetDB(db *gorm.DB) *gorm.DB {
	return db
}