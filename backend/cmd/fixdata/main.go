// fixdata cross-checks the devices table against 中联重科数据中心台账.xlsx
// and prints / optionally applies corrections. Focus cases:
//   1. device_type is empty in DB but present in xlsx → fill.
//   2. serial_number in DB holds a device_type value (e.g. "服务器") → swap with xlsx's real SN, and fix device_type too.
//   3. Any other xlsx string field where DB value is blank and xlsx has a value → fill (safe).
// Non-empty DB values that disagree with xlsx are reported but NOT auto-overwritten
// (they may have been edited in-system after import). Use -force to overwrite those too.
package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type xlsxRow struct {
	Sheet  string
	RowNum int // 1-indexed excel row number, for diagnostics

	AssetNumber     string
	Status          string
	Datacenter      string
	Cabinet         string
	UPosition       string
	Brand           string
	Model           string
	DeviceType      string
	SerialNumber    string
	OS              string
	IPAddress       string
	SystemAccount   string
	MgmtIP          string
	MgmtAccount     string
	Purpose         string
	Owner           string
	Remark          string
	Vendor          string
	ContractNo      string
	FinanceNo       string
	StorageLocation string
	Custodian       string
	ManufactureDate *time.Time
	WarrantyStart   *time.Time
	WarrantyEnd     *time.Time
	ArrivalDate     *time.Time
}

type dbDevice struct {
	ID              uint
	Source          string
	AssetNumber     string     `gorm:"column:asset_number"`
	Status          string     `gorm:"column:status"`
	Datacenter      string     `gorm:"column:datacenter"`
	Cabinet         string     `gorm:"column:cabinet"`
	UPosition       string     `gorm:"column:u_position"`
	Brand           string     `gorm:"column:brand"`
	Model           string     `gorm:"column:model"`
	DeviceType      string     `gorm:"column:device_type"`
	SerialNumber    string     `gorm:"column:serial_number"`
	OS              string     `gorm:"column:os"`
	IPAddress       string     `gorm:"column:ip_address"`
	SystemAccount   string     `gorm:"column:system_account"`
	MgmtIP          string     `gorm:"column:mgmt_ip"`
	MgmtAccount     string     `gorm:"column:mgmt_account"`
	Purpose         string     `gorm:"column:purpose"`
	Owner           string     `gorm:"column:owner"`
	Remark          string     `gorm:"column:remark"`
	Vendor          string     `gorm:"column:vendor"`
	ContractNo      string     `gorm:"column:contract_no"`
	FinanceNo       string     `gorm:"column:finance_no"`
	StorageLocation string     `gorm:"column:storage_location"`
	Custodian       string     `gorm:"column:custodian"`
	ManufactureDate *time.Time `gorm:"column:manufacture_date"`
	WarrantyStart   *time.Time `gorm:"column:warranty_start"`
	WarrantyEnd     *time.Time `gorm:"column:warranty_end"`
	ArrivalDate     *time.Time `gorm:"column:arrival_date"`
}

func (dbDevice) TableName() string { return "devices" }

type fieldDiff struct {
	Column string
	Old    string
	New    string
	Safe   bool // true → safe to auto-apply; false → risky overwrite (requires -force)
	Reason string
}

type pendingUpdate struct {
	DeviceID  uint
	Sheet     string
	RowNum    int
	MatchedBy string
	Diffs     []fieldDiff
}

type stats struct {
	totalDb         int
	totalXlsx       int
	sourceUnknown   int
	unmatched       int
	matchedBy       map[string]int
	safeFieldCount  map[string]int
	riskyFieldCount map[string]int
}

func newStats() *stats {
	return &stats{
		matchedBy:       map[string]int{},
		safeFieldCount:  map[string]int{},
		riskyFieldCount: map[string]int{},
	}
}

func main() {
	var (
		dbPath      string
		xlsxPath    string
		apply       bool
		force       bool
		sampleLimit int
	)

	flag.StringVar(&dbPath, "db", "dc_manager.db", "SQLite database path")
	flag.StringVar(&xlsxPath, "xlsx", "../中联重科数据中心台账.xlsx", "Excel ledger path")
	flag.BoolVar(&apply, "apply", false, "apply updates to database")
	flag.BoolVar(&force, "force", false, "with -apply, also overwrite non-empty DB values that disagree with xlsx")
	flag.IntVar(&sampleLimit, "sample", 50, "number of sample updates to print per section")
	flag.Parse()

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	absXlsx, _ := filepath.Abs(xlsxPath)
	fmt.Printf("DB:   %s\n", dbPath)
	fmt.Printf("Xlsx: %s\n\n", absXlsx)

	rows, err := loadXlsx(xlsxPath)
	if err != nil {
		log.Fatalf("read xlsx: %v", err)
	}

	var devices []dbDevice
	if err := db.Find(&devices).Error; err != nil {
		log.Fatalf("load devices: %v", err)
	}

	knownTypes := buildKnownDeviceTypes(rows)
	fmt.Printf("Known device_type values from xlsx (%d): %s\n\n",
		len(knownTypes), strings.Join(sortedKeys(knownTypes), "、"))

	updates, st := matchAndDiff(devices, rows, knownTypes)
	st.totalDb = len(devices)
	st.totalXlsx = len(rows)

	printReport(st, updates, sampleLimit)

	if !apply {
		fmt.Println("\nDry run only. Re-run with -apply to write changes.")
		if !force {
			fmt.Println("(Use -apply -force to also overwrite non-empty mismatches.)")
		}
		return
	}

	applied, err := applyUpdates(db, updates, force)
	if err != nil {
		log.Fatalf("apply: %v", err)
	}
	fmt.Printf("\nApplied %d column updates across %d devices.\n", applied.cols, applied.devices)
}

// ---------- xlsx loading ----------

func loadXlsx(path string) ([]xlsxRow, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var rows []xlsxRow
	for _, sheet := range f.GetSheetList() {
		sheetRows, err := f.GetRows(sheet)
		if err != nil || len(sheetRows) < 2 {
			continue
		}
		header := sheetRows[0]
		colIdx := map[string]int{}
		for i, h := range header {
			key := strings.TrimSpace(strings.ReplaceAll(h, "\n", ""))
			if key != "" {
				colIdx[key] = i
			}
		}

		get := func(row []string, names ...string) string {
			for _, n := range names {
				if i, ok := colIdx[n]; ok && i < len(row) {
					v := strings.TrimSpace(row[i])
					if v != "" && !strings.Contains(v, "=ROW()") {
						return v
					}
				}
			}
			return ""
		}
		getDate := func(row []string, names ...string) *time.Time {
			for _, n := range names {
				if i, ok := colIdx[n]; ok && i < len(row) {
					v := strings.TrimSpace(row[i])
					if t := parseExcelDate(v); t != nil {
						return t
					}
				}
			}
			return nil
		}

		sheetCount := 0
		for i, row := range sheetRows {
			if i == 0 || allEmpty(row) {
				continue
			}
			brand := get(row, "设备品牌", "设备\n品牌")
			model := get(row, "设备型号")
			sn := get(row, "序列号")
			ip := get(row, "IP地址")
			if brand == "" && model == "" && sn == "" && ip == "" {
				continue
			}

			rows = append(rows, xlsxRow{
				Sheet:           sheet,
				RowNum:          i + 1,
				AssetNumber:     get(row, "资产编号"),
				Status:          get(row, "状态"),
				Datacenter:      get(row, "机房"),
				Cabinet:         get(row, "机柜号", "新机柜号"),
				UPosition:       get(row, "U位置", "设备位置（U数）", "设备位置\n（U数）"),
				Brand:           brand,
				Model:           model,
				DeviceType:      get(row, "设备类型"),
				SerialNumber:    sn,
				OS:              get(row, "操作系统"),
				IPAddress:       ip,
				SystemAccount:   get(row, "系统账号密码"),
				MgmtIP:          get(row, "远程管理IP"),
				MgmtAccount:     get(row, "管理口账号"),
				Purpose:         get(row, "设备用途"),
				Owner:           get(row, "责任人"),
				Remark:          get(row, "备注说明", "描述", "备注"),
				Vendor:          get(row, "厂商"),
				ContractNo:      get(row, "合同号"),
				FinanceNo:       get(row, "财务编号"),
				StorageLocation: get(row, "存放位置"),
				Custodian:       firstNonEmpty(get(row, "责任人"), get(row, "保管员")),
				ManufactureDate: getDate(row, "设备出厂时间"),
				WarrantyStart:   getDate(row, "维保起始时间"),
				WarrantyEnd:     getDate(row, "维保结束时间"),
				ArrivalDate:     getDate(row, "到货日期"),
			})
			sheetCount++
		}
		fmt.Printf("  [%s] %d rows\n", sheet, sheetCount)
	}
	fmt.Printf("Total xlsx rows: %d\n\n", len(rows))
	return rows, nil
}

func parseExcelDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range []string{
		"2006-01-02", "2006/01/02", "2006/1/2", "2006-1-2",
		"01/02/2006", "2006-01-02 15:04:05", "2006-01-02T15:04:05Z",
		"2006.01.02", "2006年01月02日",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	// Excel serial date (days since 1899-12-30).
	if n, err := strconv.ParseFloat(s, 64); err == nil && n > 1 && n < 80000 {
		base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
		t := base.Add(time.Duration(n*86400) * time.Second)
		return &t
	}
	return nil
}

func allEmpty(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func buildKnownDeviceTypes(rows []xlsxRow) map[string]bool {
	set := map[string]bool{}
	for _, r := range rows {
		if v := strings.TrimSpace(r.DeviceType); v != "" {
			set[normKey(v)] = true
		}
	}
	return set
}

// ---------- matching ----------

type xlsxIndex struct {
	rows    []*xlsxRow
	bySN    map[string][]*xlsxRow
	byLoc   map[string][]*xlsxRow
	byAsset map[string][]*xlsxRow
	byIP    map[string][]*xlsxRow
}

func newIndex() *xlsxIndex {
	return &xlsxIndex{
		bySN:    map[string][]*xlsxRow{},
		byLoc:   map[string][]*xlsxRow{},
		byAsset: map[string][]*xlsxRow{},
		byIP:    map[string][]*xlsxRow{},
	}
}

func normKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func locKey(dc, cab, u string) string {
	return normKey(dc) + "|" + normKey(cab) + "|" + normKey(u)
}

func matchAndDiff(devices []dbDevice, rows []xlsxRow, knownTypes map[string]bool) ([]pendingUpdate, *stats) {
	bySheet := map[string]*xlsxIndex{}
	for i := range rows {
		r := &rows[i]
		idx, ok := bySheet[r.Sheet]
		if !ok {
			idx = newIndex()
			bySheet[r.Sheet] = idx
		}
		idx.rows = append(idx.rows, r)
		if r.SerialNumber != "" && !knownTypes[normKey(r.SerialNumber)] {
			k := normKey(r.SerialNumber)
			idx.bySN[k] = append(idx.bySN[k], r)
		}
		if r.Datacenter != "" || r.Cabinet != "" || r.UPosition != "" {
			k := locKey(r.Datacenter, r.Cabinet, r.UPosition)
			idx.byLoc[k] = append(idx.byLoc[k], r)
		}
		if r.AssetNumber != "" {
			k := normKey(r.AssetNumber)
			idx.byAsset[k] = append(idx.byAsset[k], r)
		}
		if r.IPAddress != "" {
			k := normKey(r.IPAddress)
			idx.byIP[k] = append(idx.byIP[k], r)
		}
	}

	st := newStats()
	var updates []pendingUpdate

	for _, d := range devices {
		idx, ok := bySheet[d.Source]
		if !ok {
			st.sourceUnknown++
			continue
		}

		var xr *xlsxRow
		matchedBy := ""
		// 1) SN (only if DB's SN is not a known type value)
		if d.SerialNumber != "" && !knownTypes[normKey(d.SerialNumber)] {
			if cand := idx.bySN[normKey(d.SerialNumber)]; len(cand) == 1 {
				xr = cand[0]
				matchedBy = "sn"
			}
		}
		// 2) full location
		if xr == nil && d.Datacenter != "" && d.Cabinet != "" && d.UPosition != "" {
			if cand := idx.byLoc[locKey(d.Datacenter, d.Cabinet, d.UPosition)]; len(cand) == 1 {
				xr = cand[0]
				matchedBy = "loc"
			}
		}
		// 3) asset_number
		if xr == nil && d.AssetNumber != "" {
			if cand := idx.byAsset[normKey(d.AssetNumber)]; len(cand) == 1 {
				xr = cand[0]
				matchedBy = "asset"
			}
		}
		// 4) ip
		if xr == nil && d.IPAddress != "" {
			if cand := idx.byIP[normKey(d.IPAddress)]; len(cand) == 1 {
				xr = cand[0]
				matchedBy = "ip"
			}
		}
		if xr == nil {
			st.unmatched++
			continue
		}
		st.matchedBy[matchedBy]++

		diffs := diffRecords(d, xr, knownTypes)
		for _, df := range diffs {
			if df.Safe {
				st.safeFieldCount[df.Column]++
			} else {
				st.riskyFieldCount[df.Column]++
			}
		}
		if len(diffs) > 0 {
			updates = append(updates, pendingUpdate{
				DeviceID: d.ID, Sheet: d.Source, RowNum: xr.RowNum,
				MatchedBy: matchedBy, Diffs: diffs,
			})
		}
	}
	return updates, st
}

// ---------- diffing ----------

type strPair struct {
	name string
	dbv  string
	xv   string
}

type datePair struct {
	name string
	dbv  *time.Time
	xv   *time.Time
}

func diffRecords(d dbDevice, x *xlsxRow, knownTypes map[string]bool) []fieldDiff {
	var diffs []fieldDiff
	snIsTypo := d.SerialNumber != "" && knownTypes[normKey(d.SerialNumber)]

	// Swap case: DB SN value is actually a device_type.
	if snIsTypo && strings.TrimSpace(x.SerialNumber) != "" &&
		!strings.EqualFold(d.SerialNumber, x.SerialNumber) {
		diffs = append(diffs, fieldDiff{
			Column: "serial_number", Old: d.SerialNumber, New: x.SerialNumber,
			Safe: true, Reason: "DB value looks like a device_type; using xlsx SN",
		})
		if d.DeviceType == "" || strings.EqualFold(d.DeviceType, d.SerialNumber) {
			if x.DeviceType != "" && d.DeviceType != x.DeviceType {
				diffs = append(diffs, fieldDiff{
					Column: "device_type", Old: d.DeviceType, New: x.DeviceType,
					Safe: true, Reason: "restore from xlsx after SN/type swap",
				})
			}
		}
	}

	strPairs := []strPair{
		{"device_type", d.DeviceType, x.DeviceType},
		{"brand", d.Brand, x.Brand},
		{"model", d.Model, x.Model},
		{"ip_address", d.IPAddress, x.IPAddress},
		{"mgmt_ip", d.MgmtIP, x.MgmtIP},
		{"os", d.OS, x.OS},
		{"datacenter", d.Datacenter, x.Datacenter},
		{"cabinet", d.Cabinet, x.Cabinet},
		{"u_position", d.UPosition, x.UPosition},
		{"purpose", d.Purpose, x.Purpose},
		{"owner", d.Owner, x.Owner},
		{"vendor", d.Vendor, x.Vendor},
		{"custodian", d.Custodian, x.Custodian},
		{"contract_no", d.ContractNo, x.ContractNo},
		{"finance_no", d.FinanceNo, x.FinanceNo},
		{"storage_location", d.StorageLocation, x.StorageLocation},
		{"asset_number", d.AssetNumber, x.AssetNumber},
		{"status", d.Status, x.Status},
		{"remark", d.Remark, x.Remark},
		{"mgmt_account", d.MgmtAccount, x.MgmtAccount},
		{"system_account", d.SystemAccount, x.SystemAccount},
	}
	if !snIsTypo {
		strPairs = append(strPairs, strPair{"serial_number", d.SerialNumber, x.SerialNumber})
	}

	seen := map[string]bool{}
	for _, df := range diffs {
		seen[df.Column] = true
	}

	for _, p := range strPairs {
		if seen[p.name] {
			continue
		}
		dbv := strings.TrimSpace(p.dbv)
		xv := strings.TrimSpace(p.xv)
		if xv == "" || strings.EqualFold(dbv, xv) {
			continue
		}
		if dbv == "" {
			diffs = append(diffs, fieldDiff{
				Column: p.name, Old: p.dbv, New: p.xv,
				Safe: true, Reason: "DB empty, fill from xlsx",
			})
		} else {
			diffs = append(diffs, fieldDiff{
				Column: p.name, Old: p.dbv, New: p.xv,
				Safe: false, Reason: "DB non-empty but differs",
			})
		}
	}

	datePairs := []datePair{
		{"manufacture_date", d.ManufactureDate, x.ManufactureDate},
		{"warranty_start", d.WarrantyStart, x.WarrantyStart},
		{"warranty_end", d.WarrantyEnd, x.WarrantyEnd},
		{"arrival_date", d.ArrivalDate, x.ArrivalDate},
	}
	for _, dp := range datePairs {
		if dp.xv == nil {
			continue
		}
		if dp.dbv == nil {
			diffs = append(diffs, fieldDiff{
				Column: dp.name, Old: "", New: dp.xv.Format("2006-01-02"),
				Safe: true, Reason: "DB empty, fill from xlsx",
			})
			continue
		}
		if !dp.dbv.Equal(*dp.xv) && dp.dbv.Format("2006-01-02") != dp.xv.Format("2006-01-02") {
			diffs = append(diffs, fieldDiff{
				Column: dp.name, Old: dp.dbv.Format("2006-01-02"), New: dp.xv.Format("2006-01-02"),
				Safe: false, Reason: "DB non-empty but differs",
			})
		}
	}

	return diffs
}

// ---------- report ----------

func printReport(st *stats, updates []pendingUpdate, sampleLimit int) {
	fmt.Println("==================== Match summary ====================")
	fmt.Printf("  DB devices:          %d\n", st.totalDb)
	fmt.Printf("  xlsx data rows:      %d\n", st.totalXlsx)
	fmt.Printf("  Matched (sn):        %d\n", st.matchedBy["sn"])
	fmt.Printf("  Matched (loc):       %d\n", st.matchedBy["loc"])
	fmt.Printf("  Matched (asset):     %d\n", st.matchedBy["asset"])
	fmt.Printf("  Matched (ip):        %d\n", st.matchedBy["ip"])
	fmt.Printf("  Unmatched:           %d\n", st.unmatched)
	fmt.Printf("  Source not in xlsx:  %d\n", st.sourceUnknown)

	fmt.Println("\n============ Safe fixes (DB empty → xlsx value) ============")
	printFieldCounts(st.safeFieldCount)

	fmt.Println("\n============ Risky mismatches (non-empty, need review) ============")
	printFieldCounts(st.riskyFieldCount)

	safeUpdates := filterUpdates(updates, true, false)
	riskyUpdates := filterUpdates(updates, false, true)
	sortUpdates(safeUpdates)
	sortUpdates(riskyUpdates)

	fmt.Printf("\n============ Sample safe updates (up to %d) ============\n", sampleLimit)
	printSamples(safeUpdates, sampleLimit, true)

	fmt.Printf("\n============ Sample risky mismatches (up to %d) ============\n", sampleLimit)
	printSamples(riskyUpdates, sampleLimit, false)
}

func printFieldCounts(m map[string]int) {
	if len(m) == 0 {
		fmt.Println("  (none)")
		return
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return m[keys[i]] > m[keys[j]] })
	for _, k := range keys {
		fmt.Printf("  %-22s %d\n", k, m[k])
	}
}

func filterUpdates(updates []pendingUpdate, wantSafe, wantRisky bool) []pendingUpdate {
	var out []pendingUpdate
	for _, u := range updates {
		var keep []fieldDiff
		for _, df := range u.Diffs {
			if (df.Safe && wantSafe) || (!df.Safe && wantRisky) {
				keep = append(keep, df)
			}
		}
		if len(keep) > 0 {
			cp := u
			cp.Diffs = keep
			out = append(out, cp)
		}
	}
	return out
}

func sortUpdates(us []pendingUpdate) {
	sort.Slice(us, func(i, j int) bool {
		if us[i].Sheet != us[j].Sheet {
			return us[i].Sheet < us[j].Sheet
		}
		return us[i].DeviceID < us[j].DeviceID
	})
}

func printSamples(us []pendingUpdate, limit int, safeMode bool) {
	if len(us) == 0 {
		fmt.Println("  (none)")
		return
	}
	for i, u := range us {
		if i >= limit {
			fmt.Printf("  ...and %d more\n", len(us)-limit)
			break
		}
		kind := "SAFE"
		if !safeMode {
			kind = "RISKY"
		}
		fmt.Printf("  [%s][%s row %d → device %d via %s]\n", kind, u.Sheet, u.RowNum, u.DeviceID, u.MatchedBy)
		for _, df := range u.Diffs {
			fmt.Printf("      %-17s %q -> %q  (%s)\n", df.Column, df.Old, df.New, df.Reason)
		}
	}
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ---------- apply ----------

type applyStats struct {
	devices int
	cols    int
}

func applyUpdates(db *gorm.DB, updates []pendingUpdate, force bool) (applyStats, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return applyStats{}, tx.Error
	}

	var rs applyStats
	for _, u := range updates {
		set := map[string]any{}
		for _, df := range u.Diffs {
			if !df.Safe && !force {
				continue
			}
			set[df.Column] = df.New
		}
		if len(set) == 0 {
			continue
		}
		if err := tx.Table("devices").Where("id = ?", u.DeviceID).Updates(set).Error; err != nil {
			tx.Rollback()
			return applyStats{}, fmt.Errorf("device %d: %w", u.DeviceID, err)
		}
		rs.devices++
		rs.cols += len(set)
	}
	if err := tx.Commit().Error; err != nil {
		return applyStats{}, err
	}
	return rs, nil
}
