package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type deviceRow struct {
	ID           uint
	Brand        string
	Model        string
	SerialNumber string
	Datacenter   string
	Cabinet      string
	IPAddress    string
	UpdatedAt    string
}

// 清理设备脏数据：
//  1. 删除 model 为空（NULL 或纯空白）的设备
//  2. 按 serial_number 去重（仅对 SN 非空且非纯空白者生效）：每组保留 ID 最大的一条，其余删除
//
// 默认 dry-run，仅打印；-apply 才实际删除。
func main() {
	var (
		dbPath string
		apply  bool
		sample int
	)
	flag.StringVar(&dbPath, "db", "dc_manager.db", "SQLite database path")
	flag.BoolVar(&apply, "apply", false, "apply deletions to database (default: dry-run)")
	flag.IntVar(&sample, "sample", 30, "number of sample rows to print per category")
	flag.Parse()

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	// ===== Step 1: empty model =====
	var emptyModelRows []deviceRow
	if err := db.Raw(`
		SELECT id, brand, model, serial_number, datacenter, cabinet, ip_address, updated_at
		FROM devices
		WHERE model IS NULL OR TRIM(model) = ''
		ORDER BY id
	`).Scan(&emptyModelRows).Error; err != nil {
		log.Fatalf("failed to query empty-model devices: %v", err)
	}

	fmt.Printf("== Step 1: devices with empty model ==\n")
	fmt.Printf("Found %d device(s) with NULL or blank model.\n", len(emptyModelRows))
	printSample(emptyModelRows, sample)

	// ===== Step 2: duplicates by serial_number =====
	type snGroup struct {
		SN    string
		Count int
		IDs   string
	}
	var groups []snGroup
	if err := db.Raw(`
		SELECT TRIM(serial_number) AS sn,
		       COUNT(*) AS count,
		       GROUP_CONCAT(id, ',') AS ids
		FROM devices
		WHERE serial_number IS NOT NULL AND TRIM(serial_number) != ''
		GROUP BY TRIM(serial_number)
		HAVING COUNT(*) > 1
		ORDER BY count DESC, sn
	`).Scan(&groups).Error; err != nil {
		log.Fatalf("failed to query duplicates: %v", err)
	}

	dupVictimIDs := make([]uint, 0)
	for _, g := range groups {
		ids := parseIDs(g.IDs)
		if len(ids) <= 1 {
			continue
		}
		// keep the largest (most recent) id
		maxID := ids[0]
		for _, id := range ids[1:] {
			if id > maxID {
				maxID = id
			}
		}
		for _, id := range ids {
			if id != maxID {
				dupVictimIDs = append(dupVictimIDs, id)
			}
		}
	}

	fmt.Printf("\n== Step 2: duplicates by serial_number ==\n")
	fmt.Printf("Found %d SN group(s) with duplicates, %d row(s) would be removed (keeping highest id per group).\n",
		len(groups), len(dupVictimIDs))

	if len(groups) > 0 {
		printShown := min(sample, len(groups))
		fmt.Printf("Sample groups (first %d):\n", printShown)
		for i := 0; i < printShown; i++ {
			g := groups[i]
			fmt.Printf("  SN=%q  count=%d  ids=[%s]\n", g.SN, g.Count, g.IDs)
		}
	}

	// ===== Apply or dry-run summary =====
	if len(emptyModelRows) == 0 && len(dupVictimIDs) == 0 {
		fmt.Println("\nNothing to clean.")
		return
	}

	if !apply {
		fmt.Println("\nDry run only. Re-run with -apply to delete the rows above.")
		fmt.Println("Tip: back up the database first, e.g.:")
		backupCmd := strings.ReplaceAll("    cp dc_manager.db dc_manager.db.bak.fixmodel.$(date +@Y@m@d-@H@M@S)", "@", "%")
		fmt.Println(backupCmd)
		return
	}

	tx := db.Begin()
	if tx.Error != nil {
		log.Fatalf("begin transaction: %v", tx.Error)
	}

	if len(emptyModelRows) > 0 {
		res := tx.Exec(`DELETE FROM devices WHERE model IS NULL OR TRIM(model) = ''`)
		if res.Error != nil {
			tx.Rollback()
			log.Fatalf("delete empty-model rows: %v", res.Error)
		}
		fmt.Printf("\nDeleted %d empty-model device(s).\n", res.RowsAffected)
	}

	if len(dupVictimIDs) > 0 {
		res := tx.Exec(`DELETE FROM devices WHERE id IN ?`, dupVictimIDs)
		if res.Error != nil {
			tx.Rollback()
			log.Fatalf("delete duplicate rows: %v", res.Error)
		}
		fmt.Printf("Deleted %d duplicate device(s) (kept highest id per SN).\n", res.RowsAffected)
	}

	if err := tx.Commit().Error; err != nil {
		log.Fatalf("commit: %v", err)
	}
	fmt.Println("\nDone.")
}

func printSample(rows []deviceRow, n int) {
	if len(rows) == 0 {
		return
	}
	limit := min(n, len(rows))
	fmt.Printf("Sample (first %d):\n", limit)
	for i := 0; i < limit; i++ {
		r := rows[i]
		fmt.Printf("  [id=%d] brand=%q model=%q sn=%q dc=%q cab=%q ip=%q updated=%s\n",
			r.ID, r.Brand, r.Model, r.SerialNumber, r.Datacenter, r.Cabinet, r.IPAddress, r.UpdatedAt)
	}
}

func parseIDs(s string) []uint {
	parts := strings.Split(s, ",")
	out := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var v uint
		_, err := fmt.Sscanf(p, "%d", &v)
		if err == nil {
			out = append(out, v)
		}
	}
	return out
}
