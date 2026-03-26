package main

import (
	"dcmanager/database"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// cabinetRe matches patterns like "A01", "B17", "A-01", "C-04", "B 03" etc.
// Group 1: letter(s), Group 2: optional separator, Group 3: digits
var cabinetRe = regexp.MustCompile(`^([A-Za-z]+)[\s\-]*(\d+)$`)

func normalizeCabinet(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	m := cabinetRe.FindStringSubmatch(raw)
	if m == nil {
		return raw // unrecognized format, leave as-is
	}
	letter := strings.ToUpper(m[1])
	num := m[2]
	// pad to at least 2 digits
	if len(num) == 1 {
		num = "0" + num
	}
	return fmt.Sprintf("%s-%s", letter, num)
}

func main() {
	dbPath := "dc_manager.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}
	database.Init(dbPath)

	// --- Devices table ---
	type Row struct {
		ID      uint
		Cabinet string
	}

	var devices []Row
	database.DB.Raw("SELECT id, cabinet FROM devices WHERE cabinet != ''").Scan(&devices)

	fmt.Printf("=== Devices: %d records with cabinet ===\n", len(devices))
	fixedCount := 0
	for _, d := range devices {
		normalized := normalizeCabinet(d.Cabinet)
		if normalized != d.Cabinet {
			fmt.Printf("  [Device %d] %q -> %q\n", d.ID, d.Cabinet, normalized)
			database.DB.Exec("UPDATE devices SET cabinet = ? WHERE id = ?", normalized, d.ID)
			fixedCount++
		}
	}
	fmt.Printf("  Fixed: %d\n\n", fixedCount)

	// --- Inspections table ---
	var inspections []Row
	database.DB.Raw("SELECT id, cabinet FROM inspections WHERE cabinet != ''").Scan(&inspections)

	fmt.Printf("=== Inspections: %d records with cabinet ===\n", len(inspections))
	fixedCount = 0
	for _, i := range inspections {
		normalized := normalizeCabinet(i.Cabinet)
		if normalized != i.Cabinet {
			fmt.Printf("  [Inspection %d] %q -> %q\n", i.ID, i.Cabinet, normalized)
			database.DB.Exec("UPDATE inspections SET cabinet = ? WHERE id = ?", normalized, i.ID)
			fixedCount++
		}
	}
	fmt.Printf("  Fixed: %d\n", fixedCount)

	fmt.Println("\nDone!")
}
