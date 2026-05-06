package handlers

import (
	"bytes"
	"dcmanager/config"
	"dcmanager/database"
	"dcmanager/models"
	"dcmanager/services"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

var (
	inspDcRe      = regexp.MustCompile(`(?i)^(IDC)?\s*(\d+)\s*[-]\s*(\d+)$`)
	inspCabinetRe = regexp.MustCompile(`^([A-Za-z]+)[\s\-]*(\d+)$`)
	inspURangeRe  = regexp.MustCompile(`^(\d+)\s*[-~]\s*(\d+)\s*[Uu]$`)
	inspUSingleRe = regexp.MustCompile(`^(\d+)\s*[Uu]$`)
)

func inspectionLifecycleService() services.InspectionLifecycleService {
	return services.InspectionLifecycleService{
		DB:            database.DB,
		WebhookSender: services.NewConfiguredInspectionWebhookSender(database.DB),
	}
}

// normalizeSeverity maps raw severity values to 严重/一般/轻微
func normalizeSeverity(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "严重", "紧急", "critical", "high", "p0", "p1":
		return "严重"
	case "轻微", "低", "low", "minor", "p3", "p4":
		return "轻微"
	default:
		return "一般"
	}
}

// normalizeInspStatus maps raw status values to 待处理/处理中/已解决
func normalizeInspStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "处理中", "进行中", "in progress", "processing":
		return "处理中"
	case "已解决", "已处理", "完成", "resolved", "closed", "done":
		return "已解决"
	default:
		return "待处理"
	}
}

func normalizeDatacenterLocal(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	m := inspDcRe.FindStringSubmatch(raw)
	if m == nil {
		return raw
	}
	return fmt.Sprintf("IDC%s-%s", m[2], m[3])
}

func normalizeCabinetLocal(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	m := inspCabinetRe.FindStringSubmatch(raw)
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

func normalizeUPositionLocal(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if m := inspURangeRe.FindStringSubmatch(raw); m != nil {
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		return fmt.Sprintf("%02d-%02dU", a, b)
	}
	if m := inspUSingleRe.FindStringSubmatch(raw); m != nil {
		a, _ := strconv.Atoi(m[1])
		return fmt.Sprintf("%02dU", a)
	}
	return raw
}

// autoMatchDeviceLocal finds a device by datacenter+cabinet+U range, returns device ID or nil
func autoMatchDeviceLocal(datacenter, cabinet, uPosition string) *uint {
	db := database.DB.Where("datacenter = ? AND cabinet = ?", datacenter, cabinet)
	startU, endU := parseUPosition(uPosition)

	if startU != nil && endU != nil {
		var devices []models.Device
		db.Where("start_u IS NOT NULL AND end_u IS NOT NULL AND start_u <= ? AND end_u >= ?", *endU, *startU).
			Find(&devices)
		if len(devices) == 1 {
			id := devices[0].ID
			return &id
		}
		if len(devices) > 1 {
			for i := range devices {
				if devices[i].StartU != nil && devices[i].EndU != nil &&
					*devices[i].StartU == *startU && *devices[i].EndU == *endU {
					id := devices[i].ID
					return &id
				}
			}
			id := devices[0].ID
			return &id
		}
	} else {
		var devices []models.Device
		db.Find(&devices)
		if len(devices) == 1 {
			id := devices[0].ID
			return &id
		}
	}
	return nil
}

var allowedInspectionOrderBy = map[string]bool{
	"id": true, "found_at": true, "resolved_at": true,
	"severity": true, "status": true, "datacenter": true, "inspector": true,
	"assignee_name": true, "escalation_level": true, "last_responded_at": true, "last_escalated_at": true,
}

func GetInspections(c *gin.Context) {
	var query models.InspectionQuery
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

	db := database.DB.Model(&models.Inspection{}).Preload("Device")
	if query.Datacenter != "" {
		db = db.Where("datacenter LIKE ?", "%"+query.Datacenter+"%")
	}
	if query.Cabinet != "" {
		db = db.Where("cabinet LIKE ?", "%"+query.Cabinet+"%")
	}
	if query.Inspector != "" {
		db = db.Where("inspector LIKE ?", "%"+query.Inspector+"%")
	}
	if query.AssigneeID > 0 {
		db = db.Where("assignee_id = ?", query.AssigneeID)
	}
	if query.Assignee != "" {
		db = db.Where("assignee_name LIKE ?", "%"+query.Assignee+"%")
	}
	if query.Severity != "" {
		db = db.Where("severity = ?", query.Severity)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Escalated == "true" {
		db = db.Where("escalation_level > 0")
	} else if query.Escalated == "false" {
		db = db.Where("escalation_level = 0")
	}
	if query.StartTime != "" {
		t, err := time.Parse("2006-01-02", query.StartTime)
		if err == nil {
			db = db.Where("found_at >= ?", t)
		}
	}
	if query.EndTime != "" {
		t, err := time.Parse("2006-01-02", query.EndTime)
		if err == nil {
			db = db.Where("found_at <= ?", t.Add(24*time.Hour))
		}
	}
	if query.Keyword != "" {
		kw := "%" + query.Keyword + "%"
		db = db.Where("datacenter LIKE ? OR cabinet LIKE ? OR inspector LIKE ? OR assignee_name LIKE ? OR issue LIKE ? OR remark LIKE ?",
			kw, kw, kw, kw, kw, kw)
	}

	var total int64
	db.Count(&total)

	orderBy := "found_at"
	if allowedInspectionOrderBy[query.OrderBy] {
		orderBy = query.OrderBy
	}
	sort := "desc"
	if query.Sort == "asc" {
		sort = "asc"
	}

	var inspections []models.Inspection
	db.Order(orderBy + " " + sort).Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&inspections)

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  query.Page,
		"data":  inspections,
	})
}

func GetInspection(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var inspection models.Inspection
	if err := database.DB.Preload("Device").Preload("Images").Preload("Events").First(&inspection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "inspection not found"})
		return
	}
	c.JSON(http.StatusOK, inspection)
}

func CreateInspection(c *gin.Context) {
	var inspection models.Inspection
	if err := c.ShouldBindJSON(&inspection); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	inspection.ID = 0
	if inspection.FoundAt.IsZero() {
		inspection.FoundAt = time.Now()
	}
	inspection.Severity = normalizeSeverity(inspection.Severity)
	inspection.Status = normalizeInspStatus(inspection.Status)
	if err := services.ApplyInspectionAssignee(database.DB, &inspection); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := database.DB.Create(&inspection).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	database.DB.Preload("Device").First(&inspection, inspection.ID)
	inspectionLifecycleService().RecordCreated(inspection, getUserID(c))
	c.JSON(http.StatusCreated, inspection)
}

func UpdateInspection(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var inspection models.Inspection
	if err := database.DB.First(&inspection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "inspection not found"})
		return
	}
	if err := c.ShouldBindJSON(&inspection); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	inspection.ID = uint(id)
	inspection.Severity = normalizeSeverity(inspection.Severity)
	inspection.Status = normalizeInspStatus(inspection.Status)
	if err := services.ApplyInspectionAssignee(database.DB, &inspection); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if inspection.Status == models.InspectionStatusResolved && inspection.ResolvedAt == nil {
		now := time.Now()
		inspection.ResolvedAt = &now
	}
	database.DB.Save(&inspection)
	database.DB.Preload("Device").First(&inspection, id)
	c.JSON(http.StatusOK, inspection)
}

func TransitionInspection(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inspection id"})
		return
	}
	var body services.InspectionTransitionRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	inspection, err := inspectionLifecycleService().Transition(uint(id), body, getUserID(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.Preload("Device").First(inspection, inspection.ID)
	c.JSON(http.StatusOK, inspection)
}

func GetInspectionEvents(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var events []models.InspectionEvent
	database.DB.Where("inspection_id = ?", id).Order("created_at desc, id desc").Find(&events)
	c.JSON(http.StatusOK, events)
}

func DeleteInspection(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	// 级联删除图片文件和记录
	var images []models.InspectionImage
	database.DB.Where("inspection_id = ?", id).Find(&images)
	for _, img := range images {
		os.Remove(filepath.Join(config.UploadDir, img.FilePath))
	}
	database.DB.Where("inspection_id = ?", id).Delete(&models.InspectionImage{})

	if err := database.DB.Delete(&models.Inspection{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func BatchDeleteInspections(c *gin.Context) {
	var body struct {
		IDs []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ids"})
		return
	}

	// 级联删除图片文件和记录
	var images []models.InspectionImage
	database.DB.Where("inspection_id IN ?", body.IDs).Find(&images)
	for _, img := range images {
		os.Remove(filepath.Join(config.UploadDir, img.FilePath))
	}
	database.DB.Where("inspection_id IN ?", body.IDs).Delete(&models.InspectionImage{})

	if err := database.DB.Delete(&models.Inspection{}, body.IDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": len(body.IDs)})
}

func ImportInspections(c *gin.Context) {
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

	var allInspections []models.Inspection

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

			inspector := getCol(row, "巡检人", "巡检员", "检查人", "操作人")
			issue := getCol(row, "问题描述", "问题", "故障描述", "描述")
			if inspector == "" && issue == "" {
				continue
			}

			datacenter := normalizeDatacenterLocal(getCol(row, "机房", "数据中心", "IDC"))
			cabinet := normalizeCabinetLocal(getCol(row, "机柜", "机柜号", "机柜编号"))
			upos := normalizeUPositionLocal(getCol(row, "U位", "U位置", "位置", "设备位置"))
			startU, endU := parseUPosition(upos)

			var foundAt time.Time
			if t := getCellDate(row, "发现时间", "巡检时间", "时间", "发现日期"); t != nil {
				foundAt = *t
			} else {
				foundAt = time.Now()
			}

			insp := models.Inspection{
				Datacenter:   datacenter,
				Cabinet:      cabinet,
				UPosition:    upos,
				StartU:       startU,
				EndU:         endU,
				FoundAt:      foundAt,
				Inspector:    inspector,
				AssigneeName: getCol(row, "责任人", "处理人", "负责人"),
				Issue:        issue,
				Severity:     normalizeSeverity(getCol(row, "等级", "严重程度", "问题等级", "级别")),
				Status:       normalizeInspStatus(getCol(row, "状态", "处理状态", "问题状态")),
				ResolvedAt:   getCellDate(row, "解决时间", "处理时间", "完成时间"),
				Remark:       getCol(row, "备注", "备注说明", "说明"),
			}

			allInspections = append(allInspections, insp)
		}
	}

	if !confirm {
		preview := allInspections
		if len(preview) > 10 {
			preview = preview[:10]
		}
		c.JSON(http.StatusOK, gin.H{
			"preview": preview,
			"count":   len(allInspections),
		})
		return
	}

	inserted := 0
	skipped := 0
	for _, insp := range allInspections {
		if insp.AssigneeName != "" {
			user, err := services.FindInspectionAssigneeByName(database.DB, insp.AssigneeName)
			if err != nil || user == nil {
				skipped++
				continue
			}
			insp.AssigneeID = &user.ID
			insp.AssigneeName = user.DisplayName
			if insp.AssigneeName == "" {
				insp.AssigneeName = user.Username
			}
		}
		if insp.Datacenter != "" && insp.Cabinet != "" {
			if deviceID := autoMatchDeviceLocal(insp.Datacenter, insp.Cabinet, insp.UPosition); deviceID != nil {
				insp.DeviceID = deviceID
			}
		}
		if err := database.DB.Create(&insp).Error; err != nil {
			skipped++
			continue
		}
		inserted++
	}

	c.JSON(http.StatusOK, gin.H{
		"inserted": inserted,
		"skipped":  skipped,
		"message":  fmt.Sprintf("导入成功，新增%d条，跳过%d条", inserted, skipped),
	})
}

func GetDashboard(c *gin.Context) {
	// 各机房问题数量
	type RoomStat struct {
		Datacenter string `json:"datacenter"`
		Count      int64  `json:"count"`
	}
	var roomStats []RoomStat
	database.DB.Model(&models.Inspection{}).
		Select("datacenter, count(*) as count").
		Where("status != ?", "已解决").
		Group("datacenter").
		Scan(&roomStats)

	// 近期未解决问题（支持分页）
	issuesPage := 1
	issuesPageSize := 20
	if p, err := strconv.Atoi(c.Query("issues_page")); err == nil && p > 0 {
		issuesPage = p
	}
	if ps, err := strconv.Atoi(c.Query("issues_page_size")); err == nil && ps > 0 && ps <= 100 {
		issuesPageSize = ps
	}

	issueDB := database.DB.Model(&models.Inspection{}).Where("status != ?", "已解决")
	var recentIssuesTotal int64
	issueDB.Count(&recentIssuesTotal)

	var recentIssues []models.Inspection
	database.DB.Preload("Device").
		Where("status != ?", "已解决").
		Order("found_at DESC").
		Offset((issuesPage - 1) * issuesPageSize).
		Limit(issuesPageSize).
		Find(&recentIssues)

	// 近30天每日问题趋势
	type TrendStat struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	var trends []TrendStat
	database.DB.Model(&models.Inspection{}).
		Select("date(found_at) as date, count(*) as count").
		Where("found_at >= date('now', '-30 days')").
		Group("date(found_at)").
		Order("date").
		Scan(&trends)

	// 问题状态总计
	type StatusStat struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusStats []StatusStat
	database.DB.Model(&models.Inspection{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&statusStats)

	// 严重等级分布
	type SeverityStat struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}
	var severityStats []SeverityStat
	database.DB.Model(&models.Inspection{}).
		Select("severity, count(*) as count").
		Where("status != ?", "已解决").
		Group("severity").
		Scan(&severityStats)

	// 设备状态分布
	type DeviceStatusStat struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var deviceStatusStats []DeviceStatusStat
	database.DB.Model(&models.Device{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&deviceStatusStats)

	// 各机房设备数量统计
	type DatacenterDeviceStat struct {
		Datacenter string `json:"datacenter"`
		Count      int64  `json:"count"`
	}
	var datacenterDeviceStats []DatacenterDeviceStat
	database.DB.Model(&models.Device{}).
		Select("datacenter, count(*) as count").
		Where("datacenter != ''").
		Group("datacenter").
		Order("count DESC").
		Scan(&datacenterDeviceStats)

	// 设备类型分布
	type DeviceTypeStat struct {
		DeviceType string `json:"device_type"`
		Count      int64  `json:"count"`
	}
	var deviceTypeStats []DeviceTypeStat
	database.DB.Model(&models.Device{}).
		Select("device_type, count(*) as count").
		Where("device_type != ''").
		Group("device_type").
		Order("count DESC").
		Scan(&deviceTypeStats)

	// 设备总数
	var totalDevices int64
	database.DB.Model(&models.Device{}).Count(&totalDevices)

	c.JSON(http.StatusOK, gin.H{
		"room_stats":              roomStats,
		"recent_issues":           recentIssues,
		"recent_issues_total":     recentIssuesTotal,
		"trends":                  trends,
		"status_stats":            statusStats,
		"severity_stats":          severityStats,
		"device_status_stats":     deviceStatusStats,
		"datacenter_device_stats": datacenterDeviceStats,
		"device_type_stats":       deviceTypeStats,
		"total_devices":           totalDevices,
	})
}
