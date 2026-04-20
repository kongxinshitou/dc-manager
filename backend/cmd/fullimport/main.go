// fullimport: clears all data and performs a full import from the Excel ledger
package main

import (
	"dcmanager/database"
	"dcmanager/models"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

var (
	uRangeRe  = regexp.MustCompile(`^(\d+)\s*[-~]\s*(\d+)\s*[Uu]$`)
	uSingleRe = regexp.MustCompile(`^(\d+)\s*[Uu]$`)
	cabinetRe = regexp.MustCompile(`^([A-Za-z]+)[\s\-]*(\d+)$`)
)

// dcNameMapping maps device-sheet datacenter names to canonical names
var dcNameMapping = map[string]string{
	"IDC1-1":       "数据中心1-1",
	"IDC1-2":       "数据中心1-2",
	"IDC2-1":       "数据中心2-1",
	"数据中心2-1":   "数据中心2-1",
	"2-1机房":       "数据中心2-1",
	"1-1机房":       "数据中心1-1",
	"1-2机房":       "数据中心1-2",
	"总部大楼负二":   "总部大楼负二",
	"高机办公楼机房": "高机办公楼机房",
	"土方办公机房":   "土方办公机房",
	"泵送办公楼机房": "泵送办公楼机房",
	"科技园4楼":     "科技园4楼",
	"科技园智能所":   "科技园智能所",
	"移动":         "移动机房",
	"仓库":         "仓库",
	"异地园区":     "异地园区",
}

// Datacenters without layout sheets
var noLayoutDatacenters = []string{
	"总部大楼负二",
	"科技园4楼",
	"科技园智能所",
	"移动机房",
	"异地园区",
	"仓库",
}

// Device sheets to import in order (detailed first, then supplement)
var deviceSheets = []string{
	"数据中心",
	"存储",
	"总部大楼",
	"高机",
	"土方",
	"泵送",
	"科技园",
	"移动",
	"异地园区",
	"国拨",
}

func parseUPosition(pos string) (startU, endU *int) {
	pos = strings.TrimSpace(pos)
	if pos == "" {
		return nil, nil
	}
	if m := uRangeRe.FindStringSubmatch(pos); m != nil {
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		return &a, &b
	}
	if m := uSingleRe.FindStringSubmatch(pos); m != nil {
		a, _ := strconv.Atoi(m[1])
		return &a, &a
	}
	return nil, nil
}

func normalizeCabinet(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	// Strip HDA/HAD prefix: "HDA F-A" → "F-A", "HAD F-A" → "F-A"
	upper := strings.ToUpper(raw)
	if strings.HasPrefix(upper, "HDA ") || strings.HasPrefix(upper, "HAD ") {
		raw = strings.TrimSpace(raw[4:])
	} else if strings.HasPrefix(upper, "HDA") || strings.HasPrefix(upper, "HAD") {
		raw = strings.TrimSpace(raw[3:])
	}
	m := cabinetRe.FindStringSubmatch(raw)
	if m == nil {
		return raw
	}
	letter := strings.ToUpper(m[1])
	num := m[2]
	if len(num) == 1 {
		num = "0" + num
	}
	return fmt.Sprintf("%s-%s", letter, num)
}

func normalizeDC(raw string) string {
	raw = strings.TrimSpace(raw)
	if mapped, ok := dcNameMapping[raw]; ok {
		return mapped
	}
	return raw
}

func getCol(row []string, colIndex map[string]int, names ...string) string {
	for _, name := range names {
		if idx, ok := colIndex[name]; ok && idx < len(row) {
			v := strings.TrimSpace(row[idx])
			if v != "" && !strings.Contains(v, "=ROW()") {
				return v
			}
		}
	}
	return ""
}

func getCellDate(row []string, colIndex map[string]int, names ...string) *time.Time {
	for _, name := range names {
		if idx, ok := colIndex[name]; ok && idx < len(row) {
			v := strings.TrimSpace(row[idx])
			if v == "" {
				continue
			}
			for _, layout := range []string{"2006-01-02", "2006/01/02", "01/02/2006", "2006-01-02 15:04:05"} {
				t, err := time.Parse(layout, v)
				if err == nil {
					return &t
				}
			}
		}
	}
	return nil
}

func main() {
	dbPath := "dc_manager.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}
	xlsxPath := "001_中联重科数据中心台账_20260226.xlsx"
	if len(os.Args) > 2 {
		xlsxPath = os.Args[2]
	}

	if _, err := os.Stat(xlsxPath); os.IsNotExist(err) {
		log.Fatalf("Excel file not found: %s", xlsxPath)
	}

	database.Init(dbPath)
	db := database.DB

	// ============================================================
	// Phase 1: Clear all data (preserve users, roles, system_configs)
	// ============================================================
	log.Println("=== Phase 1: Clearing all existing data ===")
	clearTables := []string{
		"device_operations",
		"inspection_images",
		"inspections",
		"approvals",
		"devices",
		"cabinets",
		"cabinet_columns",
		"cabinet_rows",
		"datacenters",
	}
	for _, t := range clearTables {
		db.Exec("DELETE FROM " + t)
	}
	// Reset auto-increment
	for _, t := range clearTables {
		db.Exec("DELETE FROM sqlite_sequence WHERE name = ?", t)
	}
	log.Println("All device, datacenter, and inspection data cleared.")

	// ============================================================
	// Phase 2: Import topology from layout sheets
	// ============================================================
	log.Println("=== Phase 2: Importing topology from layout sheets ===")

	f, err := excelize.OpenFile(xlsxPath)
	if err != nil {
		log.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Build lookup maps
	dcNameToID := make(map[string]uint)              // canonical dc name -> dc ID
	cabinetNameToID := make(map[uint]map[string]uint) // dc ID -> cabinet name -> cabinet ID

	for sheetName, dcName := range database.LayoutSheetConfig {
		idx, err := f.GetSheetIndex(sheetName)
		if err != nil || idx == -1 {
			log.Printf("  Sheet [%s] not found, skipping", sheetName)
			continue
		}
		database.ImportLayoutSheet(db, f, sheetName, dcName)
	}

	// Parse remote site layout sheets (simple format without "机柜编号：" prefix)
	simpleLayoutSheets := map[string]string{
		"高机落位": "高机办公楼机房",
		"土方落位": "土方办公机房",
		"泵送落位": "泵送办公楼机房",
	}
	for sheetName, dcName := range simpleLayoutSheets {
		idx, err := f.GetSheetIndex(sheetName)
		if err != nil || idx == -1 {
			continue
		}
		// Delete the empty datacenter created by ImportLayoutSheet (it parsed 0 cabinets)
		db.Where("name = ?", dcName).Delete(&models.Datacenter{})
		parseSimpleLayoutSheet(db, f, sheetName, dcName)
	}

	// Load all datacenter and cabinet lookups
	var datacenters []models.Datacenter
	db.Find(&datacenters)
	for _, dc := range datacenters {
		dcNameToID[dc.Name] = dc.ID
		cabinetNameToID[dc.ID] = make(map[string]uint)
		var cabinets []models.Cabinet
		db.Where("datacenter_id = ?", dc.ID).Find(&cabinets)
		for _, cab := range cabinets {
			cabinetNameToID[dc.ID][cab.Name] = cab.ID
		}
		log.Printf("  Datacenter [%s]: %d cabinets", dc.Name, len(cabinets))
	}

	// Create Datacenter records for locations without layout sheets
	for _, dcName := range noLayoutDatacenters {
		if _, exists := dcNameToID[dcName]; exists {
			continue
		}
		dc := models.Datacenter{Name: dcName, MaxU: 47, CurrentStatus: "运行中"}
		if err := db.Create(&dc).Error; err != nil {
			log.Printf("  Failed to create datacenter %s: %v", dcName, err)
			continue
		}
		dcNameToID[dcName] = dc.ID
		cabinetNameToID[dc.ID] = make(map[string]uint)
		log.Printf("  Created datacenter: %s (no layout)", dcName)
	}

	log.Printf("Topology: %d datacenters created", len(dcNameToID))

	// ============================================================
	// Phase 3: Import devices from flat table sheets
	// ============================================================
	log.Println("=== Phase 3: Importing devices ===")

	seenSerials := make(map[string]bool)
	totalImported := 0
	totalSkipped := 0

	// Helper: parse standard device sheets (same column format)
	parseStandardSheet := func(sheetName string) {
		rows, err := f.GetRows(sheetName)
		if err != nil || len(rows) < 2 {
			return
		}

		header := rows[0]
		colIndex := make(map[string]int)
		for i, h := range header {
			h = strings.TrimSpace(strings.ReplaceAll(h, "\n", ""))
			colIndex[h] = i
		}

		count := 0
		for i, row := range rows {
			if i == 0 || len(row) == 0 {
				continue
			}
			allEmpty := true
			for _, cell := range row {
				if strings.TrimSpace(cell) != "" {
					allEmpty = false
					break
				}
			}
			if allEmpty {
				continue
			}

			uPos := getCol(row, colIndex, "U位置", "设备位置（U数）", "设备位置\n（U数）")
			startU, endU := parseUPosition(uPos)
			serial := getCol(row, colIndex, "序列号")

			// Dedup by serial number
			if serial != "" {
				if seenSerials[serial] {
					totalSkipped++
					continue
				}
				seenSerials[serial] = true
			}

			dcRaw := getCol(row, colIndex, "机房")
			if dcRaw == "" {
				// Fallback: use sheet name as datacenter for specific sheets
				if sheetName == "异地园区" || sheetName == "科技园" {
					dcRaw = sheetName
				}
			}
			dcNorm := normalizeDC(dcRaw)
			cabRaw := getCol(row, colIndex, "机柜号", "新机柜号")
			cabNorm := normalizeCabinet(cabRaw)

			device := models.Device{
				Source:          sheetName,
				AssetNumber:     getCol(row, colIndex, "资产编号"),
				Status:          "out_stock",
				Datacenter:      dcNorm,
				Cabinet:         cabNorm,
				UPosition:       uPos,
				StartU:          startU,
				EndU:            endU,
				Brand:           getCol(row, colIndex, "设备品牌", "设备\n品牌"),
				Model:           getCol(row, colIndex, "设备型号"),
				DeviceType:      getCol(row, colIndex, "设备类型"),
				SerialNumber:    serial,
				OS:              getCol(row, colIndex, "操作系统"),
				IPAddress:       getCol(row, colIndex, "IP地址"),
				SystemAccount:   getCol(row, colIndex, "系统账号密码"),
				MgmtIP:          getCol(row, colIndex, "远程管理IP"),
				MgmtAccount:     getCol(row, colIndex, "管理口账号"),
				Purpose:         getCol(row, colIndex, "设备用途"),
				Owner:           getCol(row, colIndex, "责任人"),
				Remark:          getCol(row, colIndex, "备注说明", "描述", "备注"),
				WarrantyStart:   getCellDate(row, colIndex, "维保起始时间"),
				WarrantyEnd:     getCellDate(row, colIndex, "维保结束时间"),
				ManufactureDate: getCellDate(row, colIndex, "设备出厂时间"),
				Vendor:          getCol(row, colIndex, "厂商"),
				ArrivalDate:     getCellDate(row, colIndex, "到货日期"),
				ContractNo:      getCol(row, colIndex, "合同号"),
				FinanceNo:       getCol(row, colIndex, "财务编号"),
				StorageLocation: getCol(row, colIndex, "存放位置"),
				Custodian:       getCol(row, colIndex, "责任人", "保管员"),
				DeviceStatus:    "out_stock",
				SubStatus:       "racked",
			}

			// Skip rows with no meaningful device data
			if device.Brand == "" && device.Model == "" && device.SerialNumber == "" && device.IPAddress == "" && device.MgmtIP == "" {
				continue
			}

			if err := db.Create(&device).Error; err != nil {
				log.Printf("  Skip row %d in [%s]: %v", i+1, sheetName, err)
				totalSkipped++
				continue
			}
			count++
		}
		log.Printf("  Sheet [%s]: imported %d devices", sheetName, count)
		totalImported += count
	}

	// Import standard sheets in order
	for _, sheet := range deviceSheets {
		if sheet == "国拨" {
			continue // handled separately
		}
		idx, err := f.GetSheetIndex(sheet)
		if err != nil || idx == -1 {
			continue
		}
		parseStandardSheet(sheet)
	}

	// Import 国拨 sheet (different column format)
	parseGuoboSheet(db, f, seenSerials, &totalImported, &totalSkipped)

	// Import 合并 sheet as supplement (basic info only)
	parseMergeSheet(db, f, seenSerials, &totalImported, &totalSkipped)

	log.Printf("Total devices imported: %d, skipped: %d", totalImported, totalSkipped)

	// ============================================================
	// Phase 4: Auto-create cabinets for ALL datacenters
	// (for locations without layout sheets AND for missing cabinets in existing layouts)
	// ============================================================
	log.Println("=== Phase 4: Auto-creating missing cabinets ===")

	cabinetCreated := 0
	// For ALL datacenters, check if devices reference cabinets not in the topology
	for dcName, dcID := range dcNameToID {
		var cabinetNames []string
		db.Model(&models.Device{}).
			Where("datacenter = ? AND cabinet != '' AND UPPER(cabinet) != 'N/A'", dcName).
			Distinct("cabinet").
			Pluck("cabinet", &cabinetNames)

		sort.Strings(cabinetNames)
		for _, cabName := range cabinetNames {
			if cabName == "" {
				continue
			}
			if _, exists := cabinetNameToID[dcID][cabName]; exists {
				continue
			}
			cabinet := models.Cabinet{
				DatacenterID: dcID,
				Name:         cabName,
				Height:       47,
				Width:        60,
				Depth:        120,
				CabinetType:  "standard",
			}
			if err := db.Create(&cabinet).Error; err != nil {
				log.Printf("  Failed to create cabinet %s/%s: %v", dcName, cabName, err)
				continue
			}
			cabinetNameToID[dcID][cabName] = cabinet.ID
			cabinetCreated++
		}
	}
	log.Printf("  Auto-created %d cabinets", cabinetCreated)

	// ============================================================
	// Phase 5: Link all devices to topology
	// ============================================================
	log.Println("=== Phase 5: Linking devices to topology ===")

	linkDevicesToTopology(db, dcNameToID, cabinetNameToID)

	log.Println("=== Full import completed ===")
}

func parseGuoboSheet(db *gorm.DB, f *excelize.File, seenSerials map[string]bool, totalImported, totalSkipped *int) {
	sheetName := "国拨"
	idx, err := f.GetSheetIndex(sheetName)
	if err != nil || idx == -1 {
		return
	}
	rows, err := f.GetRows(sheetName)
	if err != nil || len(rows) < 2 {
		return
	}

	header := rows[0]
	colIndex := make(map[string]int)
	for i, h := range header {
		h = strings.TrimSpace(strings.ReplaceAll(h, "\n", ""))
		colIndex[h] = i
	}

	count := 0
	for i, row := range rows {
		if i == 0 || len(row) == 0 {
			continue
		}
		allEmpty := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			continue
		}

		dcRaw := getCol(row, colIndex, "机房")
		dcNorm := normalizeDC(dcRaw)
		cabRaw := getCol(row, colIndex, "机柜")
		cabNorm := normalizeCabinet(cabRaw)
		uPos := getCol(row, colIndex, "U位")
		startU, endU := parseUPosition(uPos)

		// 国拨 sheet may have serial number in column 8 or embedded in 备注
		serial := ""
		if idx, ok := colIndex["序列号"]; ok && idx < len(row) {
			serial = strings.TrimSpace(row[idx])
		}

		if serial != "" {
			if seenSerials[serial] {
				*totalSkipped++
				continue
			}
			seenSerials[serial] = true
		}

		device := models.Device{
			Source:       sheetName,
			Datacenter:   dcNorm,
			Cabinet:      cabNorm,
			UPosition:    uPos,
			StartU:       startU,
			EndU:         endU,
			DeviceType:   getCol(row, colIndex, "设备"),
			IPAddress:    getCol(row, colIndex, "系统IP"),
			MgmtIP:       getCol(row, colIndex, "设备IP"),
			Remark:       getCol(row, colIndex, "备注"),
			SerialNumber: serial,
			DeviceStatus: "out_stock",
			SubStatus:    "racked",
		}

		if device.DeviceType == "" && device.SerialNumber == "" && device.IPAddress == "" && device.MgmtIP == "" {
			continue
		}

		if err := db.Create(&device).Error; err != nil {
			log.Printf("  Skip row %d in [%s]: %v", i+1, sheetName, err)
			*totalSkipped++
			continue
		}
		count++
	}
	log.Printf("  Sheet [%s]: imported %d devices", sheetName, count)
	*totalImported += count
}

func parseMergeSheet(db *gorm.DB, f *excelize.File, seenSerials map[string]bool, totalImported, totalSkipped *int) {
	sheetName := "合并"
	idx, err := f.GetSheetIndex(sheetName)
	if err != nil || idx == -1 {
		return
	}
	rows, err := f.GetRows(sheetName)
	if err != nil || len(rows) < 2 {
		return
	}

	header := rows[0]
	colIndex := make(map[string]int)
	for i, h := range header {
		h = strings.TrimSpace(strings.ReplaceAll(h, "\n", ""))
		colIndex[h] = i
	}

	count := 0
	for i, row := range rows {
		if i == 0 || len(row) == 0 {
			continue
		}
		allEmpty := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			continue
		}

		serial := getCol(row, colIndex, "序列号")
		if serial == "" {
			continue // 合并 sheet without serial is not useful
		}
		if seenSerials[serial] {
			*totalSkipped++
			continue
		}
		seenSerials[serial] = true

		dcRaw := getCol(row, colIndex, "机房")
		dcNorm := normalizeDC(dcRaw)
		cabRaw := getCol(row, colIndex, "新机柜号")
		cabNorm := normalizeCabinet(cabRaw)
		uPos := getCol(row, colIndex, "U位置")
		startU, endU := parseUPosition(uPos)

		device := models.Device{
			Source:       sheetName,
			Datacenter:   dcNorm,
			Cabinet:      cabNorm,
			UPosition:    uPos,
			StartU:       startU,
			EndU:         endU,
			DeviceType:   getCol(row, colIndex, "设备类型"),
			Brand:        getCol(row, colIndex, "设备品牌", "设备\n品牌"),
			Model:        getCol(row, colIndex, "设备型号"),
			SerialNumber: serial,
			AssetNumber:  getCol(row, colIndex, "资产编号"),
			DeviceStatus: "out_stock",
			SubStatus:    "racked",
		}

		if err := db.Create(&device).Error; err != nil {
			log.Printf("  Skip row %d in [%s]: %v", i+1, sheetName, err)
			*totalSkipped++
			continue
		}
		count++
	}
	log.Printf("  Sheet [%s]: imported %d devices (supplement)", sheetName, count)
	*totalImported += count
}

func linkDevicesToTopology(db *gorm.DB, dcNameToID map[string]uint, cabinetNameToID map[uint]map[string]uint) {
	// Load all devices that have datacenter+cabinet but no FK links
	var devices []models.Device
	db.Where("datacenter != '' AND datacenter_id IS NULL").Find(&devices)

	linked := 0
	noDC := 0
	noCab := 0

	for _, d := range devices {
		dcID, ok := dcNameToID[d.Datacenter]
		if !ok {
			noDC++
			continue
		}

		cabID, cabOK := cabinetNameToID[dcID][d.Cabinet]
		if !cabOK || d.Cabinet == "" {
			// Only set datacenter_id, no cabinet
			db.Model(&models.Device{}).Where("id = ?", d.ID).Update("datacenter_id", dcID)
			noCab++
			continue
		}

		db.Model(&models.Device{}).Where("id = ?", d.ID).Updates(map[string]interface{}{
			"datacenter_id": dcID,
			"cabinet_id":    cabID,
		})
		linked++
	}

	log.Printf("  Linked: %d, DC-only: %d, No DC match: %d", linked, noCab, noDC)

	// Print unmatched datacenter names
	var unmatchedDCs []string
	db.Model(&models.Device{}).
		Where("datacenter_id IS NULL AND datacenter != ''").
		Distinct("datacenter").
		Pluck("datacenter", &unmatchedDCs)
	if len(unmatchedDCs) > 0 {
		log.Printf("  Unmatched datacenter names: %v", unmatchedDCs)
	}

	// Print unmatched cabinet names (devices with dcID but no cabID)
	type UnmatchedCab struct {
		Datacenter string
		Cabinet    string
		Count      int
	}
	var unmatched []UnmatchedCab
	db.Raw(`SELECT datacenter, cabinet, count(*) as count FROM devices
		WHERE datacenter_id IS NOT NULL AND cabinet_id IS NULL AND cabinet != ''
		GROUP BY datacenter, cabinet`).Scan(&unmatched)
	if len(unmatched) > 0 {
		log.Printf("  Unmatched cabinets:")
		for _, u := range unmatched {
			log.Printf("    %s/%s (%d devices)", u.Datacenter, u.Cabinet, u.Count)
		}
	}

	// Print summary statistics
	var totalDevices int64
	db.Model(&models.Device{}).Count(&totalDevices)
	var linkedDevices int64
	db.Model(&models.Device{}).Where("cabinet_id IS NOT NULL").Count(&linkedDevices)
	var dcOnlyDevices int64
	db.Model(&models.Device{}).Where("datacenter_id IS NOT NULL AND cabinet_id IS NULL").Count(&dcOnlyDevices)
	var unlinkedDevices int64
	db.Model(&models.Device{}).Where("datacenter_id IS NULL").Count(&unlinkedDevices)

	log.Printf("  Summary: %d total devices, %d linked to cabinet, %d DC-only, %d unlinked",
		totalDevices, linkedDevices, dcOnlyDevices, unlinkedDevices)
}

// parseSimpleLayoutSheet handles layout sheets that use simple cabinet names
// without the "机柜编号：" prefix. These sheets have cabinet names in header cells
// followed by U-position rows.
func parseSimpleLayoutSheet(db *gorm.DB, f *excelize.File, sheetName, dcName string) {
	rows, err := f.GetRows(sheetName)
	if err != nil || len(rows) < 5 {
		return
	}

	// Create datacenter
	dc := models.Datacenter{Name: dcName, MaxU: 47, CurrentStatus: "运行中"}
	if err := db.Create(&dc).Error; err != nil {
		// May already exist from ImportLayoutSheet
		db.Where("name = ?", dcName).First(&dc)
	}

	// Find cabinet headers by scanning rows for cells that look like cabinet names
	// Cabinet names: A1, A2, B07, R04, PD01, UPS01, etc. - NOT numbers, not "门"/"走廊"/"电池间" etc.
	skipWords := map[string]bool{
		"门": true, "走廊": true, "电池间": true, "UPS": true, "PDF": true,
	}

	type cabInfo struct {
		name      string
		col       int
		headerRow int
	}
	var cabinets []cabInfo
	uPosRe := regexp.MustCompile(`^\d+U$`)

	// Scan rows to find cabinet headers
	for r := 0; r < min(len(rows), 15); r++ {
		for c := 0; c < len(rows[r]); c++ {
			cell := strings.TrimSpace(rows[r][c])
			if cell == "" || uPosRe.MatchString(cell) {
				continue
			}
			// Skip known non-cabinet text
			if skipWords[cell] || strings.Contains(cell, "交换机") || strings.Contains(cell, "配电柜") {
				continue
			}
			// Skip if this looks like a serial number (contains digits and letters mixed)
			// Cabinet names are typically short: A1, B07, R04, PD01, etc.
			if len(cell) > 8 {
				continue
			}
			// Check if next row at this column has a U position like "42U" or "47U"
			if r+1 < len(rows) && c < len(rows[r+1]) {
				nextCell := strings.TrimSpace(rows[r+1][c])
				if uPosRe.MatchString(nextCell) || strings.Contains(nextCell, "U") {
					// Normalize cabinet name
					cabName := normalizeCabinet(cell)
					if cabName == "" {
						cabName = cell
					}
					// Check not already found
					found := false
					for _, existing := range cabinets {
						if existing.name == cabName {
							found = true
							break
						}
					}
					if !found {
						cabinets = append(cabinets, cabInfo{name: cabName, col: c, headerRow: r})
					}
				}
			}
		}
	}

	// Collect column/row prefixes
	colSet := make(map[string]bool)
	rowSet := make(map[string]bool)
	for _, cab := range cabinets {
		colPrefix, rowPrefix := database.ParseCabinetPosition(cab.name)
		colSet[colPrefix] = true
		rowSet[rowPrefix] = true
	}

	// Create columns
	columns := database.SortedKeys(colSet)
	for i, colName := range columns {
		db.Create(&models.CabinetColumn{
			DatacenterID: dc.ID,
			Name:         colName,
			SortOrder:    i,
			ColumnType:   "standard",
		})
	}

	// Create rows
	rowNames := database.SortedKeys(rowSet)
	for i, rowName := range rowNames {
		db.Create(&models.CabinetRow{
			DatacenterID: dc.ID,
			Name:         rowName,
			SortOrder:    i,
		})
	}

	// Load column/row maps
	colMap := database.LoadColumnMap(db, dc.ID)
	rowMap := database.LoadRowMap(db, dc.ID)

	// Create cabinets
	for _, cab := range cabinets {
		colPrefix, rowPrefix := database.ParseCabinetPosition(cab.name)
		colID := colMap[colPrefix]
		rowID := rowMap[rowPrefix]

		var colIDPtr, rowIDPtr *uint
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
			Name:         cab.name,
			Height:       47,
			Width:        60,
			Depth:        120,
			CabinetType:  "standard",
		}
		db.Create(&cabinet)
	}

	log.Printf("  Simple layout: [%s] → %s: %d cabinets", sheetName, dcName, len(cabinets))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
