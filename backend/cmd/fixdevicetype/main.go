package main

import (
	"flag"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type deviceRow struct {
	ID           uint
	Source       string
	DeviceType   string
	Model        string
	Brand        string
	SerialNumber string
	Remark       string
}

type pendingUpdate struct {
	ID       uint
	OldType  string
	NewType  string
	OldModel string
	NewModel string
	OldSN    string
	NewSN    string
	OldBrand string
	NewBrand string
	Reason   string
}

var (
	reJ9      = regexp.MustCompile(`^J[0-9][0-9A-Z]+$`)
	reSRModel = regexp.MustCompile(`SR\d{3}[vV]\d`)

	canonicalTypes = map[string]bool{
		"服务器": true, "服务器_刀": true, "存储": true, "小型机": true,
		"SAN交换机": true, "数据库": true, "SAN": true, "NAS": true,
		"IB交换机": true, "F5": true, "图形工作站": true, "台式机": true,
		"软件": true, "USB": true, "税控机": true,
	}
)

func main() {
	var (
		dbPath string
		apply  bool
	)
	flag.StringVar(&dbPath, "db", "dc_manager.db", "SQLite database path")
	flag.BoolVar(&apply, "apply", false, "apply updates to database")
	flag.Parse()

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	var devices []deviceRow
	if err := db.Raw(`SELECT id, source, device_type, model, brand, serial_number, remark
		FROM devices WHERE TRIM(IFNULL(device_type, '')) != ''`).Scan(&devices).Error; err != nil {
		log.Fatalf("failed to load devices: %v", err)
	}

	var updates []pendingUpdate
	skipped := 0
	for _, d := range devices {
		dt := strings.TrimSpace(d.DeviceType)
		if canonicalTypes[dt] {
			continue
		}

		u := pendingUpdate{
			ID:       d.ID,
			OldType:  dt,
			OldModel: d.Model,
			OldSN:    d.SerialNumber,
			OldBrand: d.Brand,
		}
		dtU := strings.ToUpper(dt)

		switch {
		case dtU == "SR650V2" || dtU == "SR650V3":
			u.NewType = "服务器"
			if d.Model == "" {
				u.NewModel = dtU
			}
			if d.Brand == "" {
				u.NewBrand = "Lenovo"
			}
			u.Reason = "Lenovo ThinkSystem model misplaced"

		case dtU == "XT680":
			u.NewType = "服务器"
			if d.Model == "" {
				u.NewModel = "XT680"
			}
			u.Reason = "XT680 model misplaced"

		case dtU == "EMC 500T":
			u.NewType = "存储"
			if d.Model == "" {
				u.NewModel = "EMC 500T"
			}
			if d.Brand == "" {
				u.NewBrand = "EMC"
			}
			u.Reason = "EMC storage model misplaced"

		case reJ9.MatchString(dt):
			u.NewType = "服务器"
			if d.SerialNumber == "" {
				u.NewSN = dt
			}
			if m := reSRModel.FindString(d.Remark); m != "" && d.Model == "" {
				u.NewModel = strings.ToUpper(m)
			}
			if d.Brand == "" {
				u.NewBrand = "Lenovo"
			}
			u.Reason = "serial number misplaced as device_type"

		default:
			fmt.Printf("SKIP id=%d device_type=%q (no rule matched)\n", d.ID, dt)
			skipped++
			continue
		}

		updates = append(updates, u)
	}

	fmt.Printf("\nLoaded %d devices with non-empty device_type\n", len(devices))
	fmt.Printf("Skipped (no rule): %d\n", skipped)
	fmt.Printf("Pending fixes: %d\n\n", len(updates))

	for _, u := range updates {
		fmt.Printf("  [id=%d] type: %q -> %q | model: %q -> %q | sn: %q -> %q | brand: %q -> %q  [%s]\n",
			u.ID,
			u.OldType, u.NewType,
			u.OldModel, finalVal(u.NewModel, u.OldModel),
			u.OldSN, finalVal(u.NewSN, u.OldSN),
			u.OldBrand, finalVal(u.NewBrand, u.OldBrand),
			u.Reason,
		)
	}

	if len(updates) == 0 {
		fmt.Println("\nNo changes needed.")
		return
	}

	if !apply {
		fmt.Println("\nDry run only. Re-run with -apply to write changes.")
		return
	}

	tx := db.Begin()
	if tx.Error != nil {
		log.Fatalf("failed to start transaction: %v", tx.Error)
	}

	for _, u := range updates {
		sets := []string{"device_type = ?"}
		args := []any{u.NewType}
		if u.NewModel != "" {
			sets = append(sets, "model = ?")
			args = append(args, u.NewModel)
		}
		if u.NewSN != "" {
			sets = append(sets, "serial_number = ?")
			args = append(args, u.NewSN)
		}
		if u.NewBrand != "" {
			sets = append(sets, "brand = ?")
			args = append(args, u.NewBrand)
		}
		args = append(args, u.ID)
		sql := fmt.Sprintf("UPDATE devices SET %s WHERE id = ?", strings.Join(sets, ", "))
		if err := tx.Exec(sql, args...).Error; err != nil {
			tx.Rollback()
			log.Fatalf("failed to update device %d: %v", u.ID, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Fatalf("failed to commit updates: %v", err)
	}

	fmt.Printf("\nApplied %d device_type fixes.\n", len(updates))
}

func finalVal(newVal, oldVal string) string {
	if newVal == "" {
		return oldVal
	}
	return newVal
}
