package handlers

import (
	"dcmanager/database"
	"dcmanager/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

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

	var total int64
	db.Count(&total)

	var devices []models.Device
	db.Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&devices)

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  query.Page,
		"data":  devices,
	})
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
