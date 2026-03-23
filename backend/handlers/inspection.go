package handlers

import (
	"dcmanager/database"
	"dcmanager/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

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
	if query.Severity != "" {
		db = db.Where("severity = ?", query.Severity)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
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
		db = db.Where("datacenter LIKE ? OR cabinet LIKE ? OR inspector LIKE ? OR issue LIKE ? OR remark LIKE ?",
			kw, kw, kw, kw, kw)
	}

	var total int64
	db.Count(&total)

	var inspections []models.Inspection
	db.Order("found_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&inspections)

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  query.Page,
		"data":  inspections,
	})
}

func GetInspection(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var inspection models.Inspection
	if err := database.DB.Preload("Device").First(&inspection, id).Error; err != nil {
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
	if err := database.DB.Create(&inspection).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	database.DB.Preload("Device").First(&inspection, inspection.ID)
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
	database.DB.Save(&inspection)
	database.DB.Preload("Device").First(&inspection, id)
	c.JSON(http.StatusOK, inspection)
}

func DeleteInspection(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := database.DB.Delete(&models.Inspection{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
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

	// 近期未解决问题（最新20条）
	var recentIssues []models.Inspection
	database.DB.Preload("Device").
		Where("status != ?", "已解决").
		Order("found_at DESC").
		Limit(20).
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

	c.JSON(http.StatusOK, gin.H{
		"room_stats":     roomStats,
		"recent_issues":  recentIssues,
		"trends":         trends,
		"status_stats":   statusStats,
		"severity_stats": severityStats,
	})
}
