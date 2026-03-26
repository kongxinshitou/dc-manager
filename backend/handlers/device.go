package handlers

import (
	"bytes"
	"dcmanager/database"
	"dcmanager/models"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

var (
	// 合法格式：XX-XYU（范围）或 XYU（单个），U后缀必须存在
	uRangeRe  = regexp.MustCompile(`^(\d+)\s*[-~]\s*(\d+)\s*[Uu]$`)
	uSingleRe = regexp.MustCompile(`^(\d+)\s*[Uu]$`)
)

// parseUPosition parses "04-05U" → (4,5), "04U" → (4,4), others → (nil,nil)
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

var allowedDeviceOrderBy = map[string]bool{
	"id": true, "asset_number": true, "status": true, "datacenter": true,
	"brand": true, "device_type": true, "ip_address": true, "owner": true,
}

func GetDevices(c *gin.Context) {
	var query models.DeviceQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	db := buildDeviceQuery(&query)

	var total int64
	db.Count(&total)

	orderBy := "id"
	if allowedDeviceOrderBy[query.OrderBy] {
		orderBy = query.OrderBy
	}
	sort := "desc"
	if query.Sort == "asc" {
		sort = "asc"
	}

	var devices []models.Device
	db.Order(orderBy + " " + sort).Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&devices)

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  query.Page,
		"data":  devices,
	})
}

// buildDeviceQuery builds a filtered GORM query from DeviceQuery (without ordering/pagination)
func buildDeviceQuery(query *models.DeviceQuery) *gorm.DB {
	db := database.DB.Model(&models.Device{})
	if query.Source != "" {
		db = db.Where("source = ?", query.Source)
	}
	if query.Status != "" {
		db = db.Where("status LIKE ?", "%"+query.Status+"%")
	}
	if query.Datacenter != "" {
		db = db.Where("datacenter LIKE ?", "%"+query.Datacenter+"%")
	}
	if query.Cabinet != "" {
		db = db.Where("cabinet LIKE ?", "%"+query.Cabinet+"%")
	}
	if query.Brand != "" {
		db = db.Where("brand LIKE ?", "%"+query.Brand+"%")
	}
	if query.Model != "" {
		db = db.Where("model LIKE ?", "%"+query.Model+"%")
	}
	if query.DeviceType != "" {
		db = db.Where("device_type LIKE ?", "%"+query.DeviceType+"%")
	}
	if query.IPAddress != "" {
		db = db.Where("ip_address LIKE ?", "%"+query.IPAddress+"%")
	}
	if query.Owner != "" {
		db = db.Where("owner LIKE ?", "%"+query.Owner+"%")
	}
	if query.Keyword != "" {
		kw := "%" + query.Keyword + "%"
		db = db.Where("datacenter LIKE ? OR cabinet LIKE ? OR brand LIKE ? OR model LIKE ? OR serial_number LIKE ? OR ip_address LIKE ? OR purpose LIKE ? OR owner LIKE ? OR remark LIKE ?",
			kw, kw, kw, kw, kw, kw, kw, kw, kw)
	}
	return db
}

func GetDevice(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var device models.Device
	if err := database.DB.First(&device, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	c.JSON(http.StatusOK, device)
}

func CreateDevice(c *gin.Context) {
	var device models.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	device.ID = 0
	device.StartU, device.EndU = parseUPosition(device.UPosition)
	if err := database.DB.Create(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, device)
}

func UpdateDevice(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var device models.Device
	if err := database.DB.First(&device, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	device.ID = uint(id)
	device.StartU, device.EndU = parseUPosition(device.UPosition)
	database.DB.Save(&device)
	c.JSON(http.StatusOK, device)
}

func DeleteDevice(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := database.DB.Delete(&models.Device{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func BatchDeleteDevices(c *gin.Context) {
	var body struct {
		IDs []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ids"})
		return
	}
	if err := database.DB.Delete(&models.Device{}, body.IDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": len(body.IDs)})
}

func GetDeviceOptions(c *gin.Context) {
	type Option struct {
		Sources     []string `json:"sources"`
		Datacenters []string `json:"datacenters"`
		DeviceTypes []string `json:"device_types"`
		Brands      []string `json:"brands"`
	}
	var sources, datacenters, deviceTypes, brands []string
	database.DB.Model(&models.Device{}).Distinct("source").Pluck("source", &sources)
	database.DB.Model(&models.Device{}).Distinct("datacenter").Where("datacenter != ''").Pluck("datacenter", &datacenters)
	database.DB.Model(&models.Device{}).Distinct("device_type").Where("device_type != ''").Pluck("device_type", &deviceTypes)
	database.DB.Model(&models.Device{}).Distinct("brand").Where("brand != ''").Pluck("brand", &brands)
	c.JSON(http.StatusOK, Option{sources, datacenters, deviceTypes, brands})
}

func ExportDevices(c *gin.Context) {
	var query models.DeviceQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := buildDeviceQuery(&query)
	var devices []models.Device
	db.Order("id desc").Find(&devices)

	f := excelize.NewFile()
	defer f.Close()
	sheet := "设备台账"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ID", "来源", "状态", "机房", "机柜", "U位", "品牌", "型号", "类型",
		"序列号", "IP地址", "管理IP", "操作系统", "用途", "责任人", "维保截止", "资产编号", "备注"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	fmtDate := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.Format("2006-01-02")
	}

	for row, d := range devices {
		vals := []any{
			d.ID, d.Source, d.Status, d.Datacenter, d.Cabinet, d.UPosition,
			d.Brand, d.Model, d.DeviceType, d.SerialNumber, d.IPAddress, d.MgmtIP,
			d.OS, d.Purpose, d.Owner, fmtDate(d.WarrantyEnd), d.AssetNumber, d.Remark,
		}
		for col, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(col+1, row+2)
			f.SetCellValue(sheet, cell, v)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", `attachment; filename="devices.xlsx"`)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

func ImportDevices(c *gin.Context) {
	confirm := c.Query("confirm") == "true"

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传Excel文件"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	f, err := excelize.OpenReader(buf)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法解析Excel文件: " + err.Error()})
		return
	}
	defer f.Close()

	var allDevices []models.Device

	for _, sheetName := range f.GetSheetList() {
		rows, err := f.GetRows(sheetName)
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
				Source:          sheetName,
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

			allDevices = append(allDevices, device)
		}
	}

	if !confirm {
		preview := allDevices
		if len(preview) > 10 {
			preview = preview[:10]
		}
		c.JSON(http.StatusOK, gin.H{
			"preview": preview,
			"count":   len(allDevices),
		})
		return
	}

	// confirm=true: insert, skip duplicates by serial_number
	inserted := 0
	skipped := 0
	for _, d := range allDevices {
		if d.SerialNumber != "" {
			var existing models.Device
			if err := database.DB.Where("serial_number = ?", d.SerialNumber).First(&existing).Error; err == nil {
				skipped++
				continue
			}
		}
		if err := database.DB.Create(&d).Error; err != nil {
			skipped++
			continue
		}
		inserted++
	}

	c.JSON(http.StatusOK, gin.H{
		"inserted": inserted,
		"skipped":  skipped,
		"message":  fmt.Sprintf("导入成功，新增%d条，跳过%d条重复", inserted, skipped),
	})
}

// GetCabinets returns distinct cabinets for a given datacenter
func GetCabinets(c *gin.Context) {
	datacenter := c.Query("datacenter")
	if datacenter == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datacenter required"})
		return
	}
	var cabinets []string
	database.DB.Model(&models.Device{}).
		Where("datacenter = ? AND cabinet != ''", datacenter).
		Distinct("cabinet").
		Order("cabinet").
		Pluck("cabinet", &cabinets)
	c.JSON(http.StatusOK, gin.H{"cabinets": cabinets})
}

// GetDeviceByLocation finds a device by datacenter + cabinet + start_u/end_u
func GetDeviceByLocation(c *gin.Context) {
	datacenter := c.Query("datacenter")
	cabinet := c.Query("cabinet")

	if datacenter == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datacenter required"})
		return
	}

	db := database.DB.Where("datacenter = ?", datacenter)
	if cabinet != "" {
		db = db.Where("cabinet = ?", cabinet)
	}

	// Accept start_u and end_u as integers for precise lookup
	startUStr := c.Query("start_u")
	endUStr := c.Query("end_u")
	if startUStr != "" && endUStr != "" {
		startU, err1 := strconv.Atoi(startUStr)
		endU, err2 := strconv.Atoi(endUStr)
		if err1 == nil && err2 == nil {
			// Find device whose U range overlaps with [startU, endU]
			db = db.Where("start_u IS NOT NULL AND start_u <= ? AND end_u >= ?", endU, startU)
		}
	} else if startUStr != "" {
		u, err := strconv.Atoi(startUStr)
		if err == nil {
			db = db.Where("start_u IS NOT NULL AND start_u <= ? AND end_u >= ?", u, u)
		}
	}

	var device models.Device
	if err := db.First(&device).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"device": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"device": device})
}
