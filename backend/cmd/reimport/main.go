// reimport: clears all devices and re-imports from Excel
package main

import (
	"dcmanager/database"
	"dcmanager/models"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

var (
	uRangeRe  = regexp.MustCompile(`^(\d+)\s*[-~]\s*(\d+)\s*[Uu]$`)
	uSingleRe = regexp.MustCompile(`^(\d+)\s*[Uu]$`)
)

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

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case float64:
		return fmt.Sprintf("%.0f", val)
	case int:
		return fmt.Sprintf("%d", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func main() {
	dbPath := "dc_manager.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}
	xlsxPath := "../../中联重科数据中心台账.xlsx"
	if len(os.Args) > 2 {
		xlsxPath = os.Args[2]
	}

	database.Init(dbPath)

	// Clear all devices
	log.Println("Clearing all existing device records...")
	if err := database.DB.Exec("DELETE FROM devices").Error; err != nil {
		log.Fatalf("failed to clear devices: %v", err)
	}
	database.DB.Exec("DELETE FROM sqlite_sequence WHERE name='devices'")
	log.Println("All devices cleared.")

	// Open Excel
	f, err := excelize.OpenFile(xlsxPath)
	if err != nil {
		log.Fatalf("open xlsx: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	total := 0

	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) < 2 {
			continue
		}

		header := rows[0]
		colIndex := map[string]int{}
		for i, h := range header {
			h = strings.TrimSpace(strings.ReplaceAll(h, "\n", ""))
			colIndex[h] = i
		}

		getCol := func(row []string, names ...string) string {
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

		getCellDate := func(row []string, names ...string) *time.Time {
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

			uPos := getCol(row, "U位置", "设备位置（U数）", "设备位置\n（U数）")
			startU, endU := parseUPosition(uPos)

			device := models.Device{
				Source:          sheet,
				AssetNumber:     getCol(row, "资产编号"),
				Status:          getCol(row, "状态"),
				Datacenter:      getCol(row, "机房"),
				Cabinet:         getCol(row, "机柜号", "新机柜号"),
				UPosition:       uPos,
				StartU:          startU,
				EndU:            endU,
				Brand:           getCol(row, "设备品牌", "设备\n品牌"),
				Model:           getCol(row, "设备型号"),
				DeviceType:      getCol(row, "设备类型"),
				SerialNumber:    getCol(row, "序列号"),
				OS:              getCol(row, "操作系统"),
				IPAddress:       getCol(row, "IP地址"),
				SystemAccount:   getCol(row, "系统账号密码"),
				MgmtIP:          getCol(row, "远程管理IP"),
				MgmtAccount:     getCol(row, "管理口账号"),
				Purpose:         getCol(row, "设备用途"),
				Owner:           getCol(row, "责任人"),
				Remark:          getCol(row, "备注说明", "描述", "备注"),
				WarrantyStart:   getCellDate(row, "维保起始时间"),
				WarrantyEnd:     getCellDate(row, "维保结束时间"),
				ManufactureDate: getCellDate(row, "设备出厂时间"),
			}

			if device.Brand == "" && device.Model == "" && device.SerialNumber == "" && device.IPAddress == "" {
				continue
			}

			if err := database.DB.Create(&device).Error; err != nil {
				log.Printf("skip row %d in %s: %v", i+1, sheet, err)
				continue
			}
			count++
		}
		log.Printf("Sheet [%s]: imported %d devices", sheet, count)
		total += count
	}
	log.Printf("Total imported: %d devices", total)
}
