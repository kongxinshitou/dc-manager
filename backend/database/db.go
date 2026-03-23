package database

import (
	"dcmanager/models"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(dsn string) {
	var err error
	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	err = DB.AutoMigrate(&models.Device{}, &models.Inspection{})
	if err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}
	log.Println("Database initialized")
}
