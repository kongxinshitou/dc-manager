package database

import (
	"dcmanager/models"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

var (
	dbURangeRe = regexp.MustCompile(`^(\d+)\s*[-~]\s*(\d+)\s*[Uu]$`)
	dbUSingleRe = regexp.MustCompile(`^(\d+)\s*[Uu]$`)
)

func parseUPos(pos string) (start, end *int) {
	pos = strings.TrimSpace(pos)
	if m := dbURangeRe.FindStringSubmatch(pos); m != nil {
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		return &a, &b
	}
	if m := dbUSingleRe.FindStringSubmatch(pos); m != nil {
		a, _ := strconv.Atoi(m[1])
		return &a, &a
	}
	return nil, nil
}

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

	// 回填 start_u/end_u：对已有 u_position 但 start_u 为空的设备自动解析
	type Row struct {
		ID        uint
		UPosition string
	}
	var rows []Row
	DB.Raw("SELECT id, u_position FROM devices WHERE u_position != '' AND start_u IS NULL").Scan(&rows)
	for _, r := range rows {
		s, e := parseUPos(r.UPosition)
		if s != nil && e != nil {
			DB.Exec("UPDATE devices SET start_u = ?, end_u = ? WHERE id = ?", *s, *e, r.ID)
		}
	}
	if len(rows) > 0 {
		log.Printf("Backfilled start_u/end_u for %d devices", len(rows))
	}

	log.Println("Database initialized")
}
