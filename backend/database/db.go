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
	dbURangeRe  = regexp.MustCompile(`^(\d+)\s*[-~]\s*(\d+)\s*[Uu]$`)
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

	err = DB.AutoMigrate(
		&models.Device{}, &models.Inspection{}, &models.Role{}, &models.User{},
		&models.InspectionImage{}, &models.InspectionEvent{}, &models.DeviceOperation{}, &models.SystemConfig{},
		&models.Approval{}, &models.Datacenter{}, &models.CabinetColumn{},
		&models.CabinetRow{}, &models.Cabinet{},
	)
	if err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	seedDefaultData()

	// 一次性迁移：为已有设备填充 device_status / sub_status
	migrateDeviceStatus()

	// 一次性迁移：修正 HDA/HAD 前缀的机柜名，并清除IDC机房中的HAD列
	migrateHdaPrefix()

	// 回填 start_u/end_u：对已有 u_position 但 start_u 为空的设备自动解析

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

	// Import datacenter layout from Excel on first startup
	importExcelDatacenterLayout(DB)
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

	// 创建默认系统配置
	seedSystemConfig()
}

func seedSystemConfig() {
	var count int64
	DB.Model(&models.SystemConfig{}).Where("`key` = ?", "default_custodians").Count(&count)
	if count == 0 {
		DB.Create(&models.SystemConfig{
			Key:   "default_custodians",
			Value: "[]",
		})
		log.Println("Created default system config: default_custodians")
	}

	DB.Model(&models.SystemConfig{}).Where("`key` = ?", "inspection_webhook_config").Count(&count)
	if count == 0 {
		DB.Create(&models.SystemConfig{
			Key:   "inspection_webhook_config",
			Value: `{"enabled":false,"webhook_url":""}`,
		})
		log.Println("Created default system config: inspection_webhook_config")
	}

	DB.Model(&models.SystemConfig{}).Where("`key` = ?", "inspection_escalation_config").Count(&count)
	if count == 0 {
		DB.Create(&models.SystemConfig{
			Key:   "inspection_escalation_config",
			Value: `{"enabled":true,"scan_interval_minutes":5,"severity_hours":{"严重":2,"一般":8,"轻微":24}}`,
		})
		log.Println("Created default system config: inspection_escalation_config")
	}
}

// migrateDeviceStatus 为已有设备填充新的 device_status / sub_status 字段（仅执行一次）
func migrateDeviceStatus() {
	var needsMigration int64
	DB.Model(&models.Device{}).Where("device_status = '' OR device_status IS NULL").Count(&needsMigration)
	if needsMigration == 0 {
		return
	}

	// 有机房+机柜+U位的设备 → 出库-上架
	result := DB.Model(&models.Device{}).
		Where("datacenter != '' AND cabinet != '' AND start_u IS NOT NULL").
		Where("device_status = '' OR device_status IS NULL").
		Updates(map[string]any{
			"device_status": "out_stock",
			"sub_status":    "racked",
		})
	if result.RowsAffected > 0 {
		log.Printf("Migrated %d devices to out_stock/racked", result.RowsAffected)
	}

	// 其余设备 → 入库-新购
	result = DB.Model(&models.Device{}).
		Where("device_status = '' OR device_status IS NULL").
		Updates(map[string]any{
			"device_status": "in_stock",
			"sub_status":    "new_purchase",
		})
	if result.RowsAffected > 0 {
		log.Printf("Migrated %d devices to in_stock/new_purchase", result.RowsAffected)
	}
}

// migrateHdaPrefix 修正 HDA/HAD 前缀的机柜名，并清除IDC机房中的HAD列
func migrateHdaPrefix() {
	// 检查是否需要迁移
	var needsMigration int64
	DB.Model(&models.Cabinet{}).Where("name LIKE 'HDA %' OR name LIKE 'HAD %' OR name LIKE 'HDA-%' OR name LIKE 'HAD-%'").Count(&needsMigration)
	if needsMigration == 0 {
		return
	}

	log.Println("Migrating HDA/HAD prefixed cabinet names...")

	// 修正 devices 表中的 cabinet 字段
	type deviceRow struct {
		ID      uint
		Cabinet string
	}
	var devices []deviceRow
	DB.Raw("SELECT id, cabinet FROM devices WHERE cabinet LIKE 'HDA %' OR cabinet LIKE 'HAD %' OR cabinet LIKE 'HDA-%' OR cabinet LIKE 'HAD-%'").Scan(&devices)
	for _, d := range devices {
		newName := stripHdaPrefix(d.Cabinet)
		if newName != d.Cabinet {
			DB.Exec("UPDATE devices SET cabinet = ? WHERE id = ?", newName, d.ID)
		}
	}
	if len(devices) > 0 {
		log.Printf("Fixed %d devices with HDA/HAD prefixed cabinet names", len(devices))
	}

	// 修正 cabinets 表中的 name 字段
	var cabinets []struct {
		ID   uint
		Name string
	}
	DB.Raw("SELECT id, name FROM cabinets WHERE name LIKE 'HDA %' OR name LIKE 'HAD %' OR name LIKE 'HDA-%' OR name LIKE 'HAD-%'").Scan(&cabinets)
	for _, c := range cabinets {
		newName := stripHdaPrefix(c.Name)
		if newName != c.Name {
			DB.Exec("UPDATE cabinets SET name = ? WHERE id = ?", newName, c.ID)
		}
	}
	if len(cabinets) > 0 {
		log.Printf("Fixed %d cabinets with HDA/HAD prefixed names", len(cabinets))
	}

	// 删除IDC/数据中心机房中名称以HDA/HAD开头的列
	result := DB.Exec(`DELETE FROM cabinet_columns WHERE name LIKE 'HDA%' OR name LIKE 'HAD%'`)
	if result.RowsAffected > 0 {
		log.Printf("Deleted %d HDA/HAD columns from datacenters", result.RowsAffected)
	}
}

func stripHdaPrefix(name string) string {
	name = strings.TrimSpace(name)
	upper := strings.ToUpper(name)
	if strings.HasPrefix(upper, "HDA") || strings.HasPrefix(upper, "HAD") {
		return strings.TrimSpace(name[3:])
	}
	return name
}
