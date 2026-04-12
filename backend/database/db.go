package database

import (
	"dcmanager/models"
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
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

	err = DB.AutoMigrate(&models.Device{}, &models.Inspection{}, &models.Role{}, &models.User{}, &models.InspectionImage{})
	if err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	seedDefaultData()

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

func seedDefaultData() {
	// 创建默认角色
	var roleCount int64
	DB.Model(&models.Role{}).Count(&roleCount)
	if roleCount == 0 {
		allPerms, _ := json.Marshal(models.AllPermissions)
		inspectorPerms, _ := json.Marshal([]string{
			models.PermDeviceRead,
			models.PermInspectionRead,
			models.PermInspectionWrite,
			models.PermDashboard,
			models.PermImageUpload,
		})

		DB.Create(&models.Role{
			Name:        "admin",
			DisplayName: "管理员",
			Permissions: string(allPerms),
			IsSystem:    true,
		})
		DB.Create(&models.Role{
			Name:        "inspector",
			DisplayName: "巡检员",
			Permissions: string(inspectorPerms),
			IsSystem:    true,
		})
		log.Println("Created default roles: admin, inspector")
	}

	// 创建默认 admin 用户
	var adminCount int64
	DB.Model(&models.User{}).Count(&adminCount)
	if adminCount == 0 {
		var adminRole models.Role
		DB.Where("name = ?", "admin").First(&adminRole)

		defaultPassword := os.Getenv("ADMIN_DEFAULT_PASSWORD")
		if defaultPassword == "" {
			defaultPassword = "admin123"
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("failed to hash admin password: %v", err)
		}

		DB.Create(&models.User{
			Username:     "admin",
			PasswordHash: string(hash),
			DisplayName:  "系统管理员",
			RoleID:       adminRole.ID,
			Status:       "active",
		})
		log.Println("Created default admin user (username: admin)")
	}
}
