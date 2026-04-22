package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type unrackRow struct {
	ID           uint
	Brand        string
	Model        string
	SerialNumber string
	Datacenter   string
	Cabinet      string
	StartU       *int
	EndU         *int
	CabinetID    *uint
	DatacenterID *uint
}

// 修复历史"下架"脏数据：sub_status='unracked' 但仍保留 cabinet_id/start_u 的设备。
// 清空其机柜/U位关联字段，使机柜视图不再显示。
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

	var rows []unrackRow
	if err := db.Raw(`
		SELECT id, brand, model, serial_number, datacenter, cabinet,
		       start_u, end_u, cabinet_id, datacenter_id
		FROM devices
		WHERE sub_status = 'unracked'
		  AND (cabinet_id IS NOT NULL OR start_u IS NOT NULL OR end_u IS NOT NULL)
	`).Scan(&rows).Error; err != nil {
		log.Fatalf("failed to load devices: %v", err)
	}

	fmt.Printf("Found %d unracked device(s) still holding cabinet/U-position data:\n\n", len(rows))
	for _, r := range rows {
		cabID := "-"
		if r.CabinetID != nil {
			cabID = fmt.Sprintf("%d", *r.CabinetID)
		}
		su, eu := "-", "-"
		if r.StartU != nil {
			su = fmt.Sprintf("%d", *r.StartU)
		}
		if r.EndU != nil {
			eu = fmt.Sprintf("%d", *r.EndU)
		}
		fmt.Printf("  [id=%d] %s %s SN=%s | dc=%q cab=%q cab_id=%s U=%s-%s\n",
			r.ID, r.Brand, r.Model, r.SerialNumber, r.Datacenter, r.Cabinet, cabID, su, eu)
	}

	if len(rows) == 0 {
		fmt.Println("\nNothing to fix.")
		return
	}

	if !apply {
		fmt.Println("\nDry run only. Re-run with -apply to clear these fields.")
		return
	}

	result := db.Exec(`
		UPDATE devices
		SET datacenter = '',
		    cabinet = '',
		    u_position = '',
		    start_u = NULL,
		    end_u = NULL,
		    u_count = NULL,
		    datacenter_id = NULL,
		    cabinet_id = NULL
		WHERE sub_status = 'unracked'
		  AND (cabinet_id IS NOT NULL OR start_u IS NOT NULL OR end_u IS NOT NULL)
	`)
	if result.Error != nil {
		log.Fatalf("failed to clear fields: %v", result.Error)
	}

	fmt.Printf("\nCleared cabinet/U-position fields on %d device(s).\n", result.RowsAffected)
}
