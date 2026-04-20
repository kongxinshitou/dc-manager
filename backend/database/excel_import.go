package database

import (
	"dcmanager/models"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

var CabinetHeaderRe = regexp.MustCompile(`机柜编号：(.+)`)

// LayoutSheetConfig defines which Excel sheets to import as datacenter layouts
var LayoutSheetConfig = map[string]string{
	"1-1机房":  "数据中心1-1",
	"1-2机房":  "数据中心1-2",
	"2-1机房":  "数据中心2-1",
	"高机落位": "高机办公楼机房",
	"土方落位": "土方办公机房",
	"泵送落位": "泵送办公楼机房",
}

// importExcelDatacenterLayout imports datacenter layout from Excel file on first startup
func importExcelDatacenterLayout(db *gorm.DB) {
	// Guard: only import if no datacenters exist
	var count int64
	db.Model(&models.Datacenter{}).Count(&count)
	if count > 0 {
		return
	}

	path := os.Getenv("EXCEL_LAYOUT_PATH")
	if path == "" {
		path = "001_中联重科数据中心台账_20260226.xlsx"
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Excel layout file not found: %s (skipping import)", path)
		return
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		log.Printf("Failed to open Excel file: %v", err)
		return
	}
	defer f.Close()

	for sheetName, dcName := range LayoutSheetConfig {
		idx, err := f.GetSheetIndex(sheetName)
		if err != nil || idx == -1 {
			continue
		}
		ImportLayoutSheet(db, f, sheetName, dcName)
	}

	log.Println("Excel datacenter layout import completed")
}

// ImportLayoutSheet imports a single layout sheet as a datacenter
func ImportLayoutSheet(db *gorm.DB, f *excelize.File, sheetName, dcName string) {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Printf("Failed to read sheet %s: %v", sheetName, err)
		return
	}

	// Create datacenter
	dc := models.Datacenter{Name: dcName}
	if err := db.Create(&dc).Error; err != nil {
		log.Printf("Failed to create datacenter %s: %v", dcName, err)
		return
	}
	log.Printf("Created datacenter: %s", dcName)

	// Find all cabinet headers and their positions
	type cabinetInfo struct {
		name      string
		cabType   string
		headerCol int // 0-based column of header
		dataCol   int // 0-based column of device data (middle of 3-col group)
		headerRow int // 0-based row of header
	}

	var cabinets []cabinetInfo
	colSet := make(map[string]bool) // unique column prefixes
	rowSet := make(map[string]bool) // unique row prefixes

	for r := 0; r < len(rows); r++ {
		for c := 0; c < len(rows[r]); c++ {
			cell := strings.TrimSpace(rows[r][c])
			m := CabinetHeaderRe.FindStringSubmatch(cell)
			if m == nil {
				continue
			}

			rawName := m[1]
			// Extract name and type: "A-01(服务器)" → name="A-01", type="服务器"
			cabName, cabType := ParseCabinetHeader(rawName)

			colPrefix, rowPrefix := ParseCabinetPosition(cabName)
			colSet[colPrefix] = true
			rowSet[rowPrefix] = true

			cabinets = append(cabinets, cabinetInfo{
				name:      cabName,
				cabType:   cabType,
				headerCol: c,
				dataCol:   c + 1, // middle column has device data
				headerRow: r,
			})
		}
	}

	// Build column type map from cabinet info
	colTypeMap := make(map[string]string)
	for _, cab := range cabinets {
		colPrefix, _ := ParseCabinetPosition(cab.name)
		if cab.cabType == "HDA" || strings.Contains(strings.ToUpper(cab.name), "HDA") || strings.Contains(strings.ToUpper(cab.name), "HAD") {
			colTypeMap[colPrefix] = "hda"
		} else if strings.Contains(strings.ToUpper(cab.name), "PDU") {
			colTypeMap[colPrefix] = "pdu"
		} else if strings.Contains(strings.ToUpper(cab.name), "空调") {
			colTypeMap[colPrefix] = "aircon"
		}
	}

	// Create columns (sorted)
	columns := SortedKeys(colSet)
	for i, colName := range columns {
		colType := "standard"
		if t, ok := colTypeMap[colName]; ok {
			colType = t
		}
		db.Create(&models.CabinetColumn{
			DatacenterID: dc.ID,
			Name:         colName,
			SortOrder:    i,
			ColumnType:   colType,
		})
	}

	// Create rows (sorted)
	rowNames := SortedKeys(rowSet)
	for i, rowName := range rowNames {
		db.Create(&models.CabinetRow{
			DatacenterID: dc.ID,
			Name:         rowName,
			SortOrder:    i,
		})
	}

	// Load column/row maps for FK assignment
	colMap := LoadColumnMap(db, dc.ID)
	rowMap := LoadRowMap(db, dc.ID)

	// Create cabinets
	for _, cab := range cabinets {
		colPrefix, rowPrefix := ParseCabinetPosition(cab.name)

		// Normalize cabinet name: strip HDA/HAD prefix
		cabName := cab.name
		upper := strings.ToUpper(cabName)
		if strings.HasPrefix(upper, "HDA") || strings.HasPrefix(upper, "HAD") {
			cabName = strings.TrimSpace(cabName[3:])
		}

		cabinetType := "standard"
		if cab.cabType == "HDA" || strings.Contains(strings.ToUpper(cab.name), "HDA") || strings.Contains(strings.ToUpper(cab.name), "HAD") {
			cabinetType = "hda"
		} else if strings.Contains(strings.ToUpper(cab.name), "PDU") {
			cabinetType = "pdu"
		} else if strings.Contains(strings.ToUpper(cab.name), "空调") {
			cabinetType = "aircon"
		}

		colID := colMap[colPrefix]
		rowID := rowMap[rowPrefix]

		var colIDPtr *uint
		var rowIDPtr *uint
		if colID > 0 {
			colIDPtr = &colID
		}
		if rowID > 0 {
			rowIDPtr = &rowID
		}

		cabinet := models.Cabinet{
			DatacenterID: dc.ID,
			ColumnID:     colIDPtr,
			RowID:        rowIDPtr,
			Name:         cabName,
			Height:       47,
			Width:        60,
			Depth:        120,
			CabinetType:  cabinetType,
		}
		db.Create(&cabinet)

		// Link existing devices to this cabinet by matching datacenter+cabinet name
		// Match both original name (with HDA prefix) and normalized name
		db.Model(&models.Device{}).
			Where("datacenter = ? AND (cabinet = ? OR cabinet = ?)", dcName, cab.name, cabName).
			Updates(map[string]any{
				"datacenter_id": dc.ID,
				"cabinet_id":    cabinet.ID,
				"cabinet":       cabName,
			})
	}

	log.Printf("  Imported %d cabinets for %s", len(cabinets), dcName)
}

// ParseCabinetHeader extracts name and type from "A-01(服务器)"
func ParseCabinetHeader(raw string) (name, cabType string) {
	idx := strings.Index(raw, "(")
	if idx == -1 {
		return strings.TrimSpace(raw), ""
	}
	name = strings.TrimSpace(raw[:idx])
	cabType = strings.TrimSpace(raw[idx+1:])
	cabType = strings.TrimSuffix(cabType, ")")
	return name, cabType
}

// ParseCabinetPosition extracts column and row prefix from cabinet name
// "A-01" → "A", "1"   "HDA A-A" → "A", "A"   "HAD F-A" → "F", "A"   "C-12" → "C", "12"
func ParseCabinetPosition(name string) (colPrefix, rowPrefix string) {
	name = strings.TrimSpace(name)

	// Handle HDA/HAD names: "HDA F-A" → col="F", row="A" (strip the HDA/HAD prefix)
	upper := strings.ToUpper(name)
	if strings.HasPrefix(upper, "HDA") || strings.HasPrefix(upper, "HAD") {
		rest := strings.TrimSpace(name[3:])
		parts := strings.SplitN(rest, "-", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		}
		return rest, ""
	}

	// Standard: "A-01" → col="A", row="1"
	parts := strings.SplitN(name, "-", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return name, ""
}

// SortedKeys returns sorted keys of a string set
func SortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple sort
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

// LoadColumnMap loads column ID by name for a datacenter
func LoadColumnMap(db *gorm.DB, dcID uint) map[string]uint {
	m := make(map[string]uint)
	var cols []models.CabinetColumn
	db.Where("datacenter_id = ?", dcID).Find(&cols)
	for _, c := range cols {
		m[c.Name] = c.ID
	}
	return m
}

// LoadRowMap loads row ID by name for a datacenter
func LoadRowMap(db *gorm.DB, dcID uint) map[string]uint {
	m := make(map[string]uint)
	var rows []models.CabinetRow
	db.Where("datacenter_id = ?", dcID).Find(&rows)
	for _, r := range rows {
		m[r.Name] = r.ID
	}
	return m
}
