package handlers

import (
	"dcmanager/auth"
	"dcmanager/database"
	"dcmanager/models"
	"dcmanager/services"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// OperateDevice executes a device state transition
func OperateDevice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device id"})
		return
	}

	var body struct {
		Operation string          `json:"operation" binding:"required"`
		Details   json.RawMessage `json:"details"`
		Remark    string          `json:"remark"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var device models.Device
	if err := database.DB.First(&device, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	// For rack operations, check U-position conflicts
	if body.Operation == services.OpRack {
		var details map[string]any
		if len(body.Details) > 0 {
			json.Unmarshal(body.Details, &details)
		}
		datacenter, _ := details["datacenter"].(string)
		cabinet, _ := details["cabinet"].(string)
		startU := intFromDetails(details, "start_u")
		uCount := intFromDetails(details, "u_count")
		if startU > 0 && uCount > 0 {
			endU := startU + uCount - 1
			if err := services.CheckUPositionConflict(database.DB, datacenter, cabinet, startU, endU, uint(id)); err != nil {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
				return
			}
		}
	}

	userID := getUserID(c)
	if err := services.ExecuteTransition(database.DB, &device, body.Operation, body.Details, userID, nil); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, device)
}

// GetDeviceOperations returns operation history for a device
func GetDeviceOperations(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	var total int64
	db := database.DB.Model(&models.DeviceOperation{}).Where("device_id = ?", id)
	db.Count(&total)

	var ops []models.DeviceOperation
	db.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&ops)

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  page,
		"data":  ops,
	})
}

// BatchUpdateCustodian batch updates custodian for devices
func BatchUpdateCustodian(c *gin.Context) {
	var body struct {
		DeviceIDs []uint `json:"device_ids" binding:"required"`
		Custodian string `json:"custodian" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := database.DB.Model(&models.Device{}).Where("id IN ?", body.DeviceIDs).
		Update("custodian", body.Custodian)

	c.JSON(http.StatusOK, gin.H{
		"updated": result.RowsAffected,
	})
}

// GetConfig returns a system config value
func GetConfig(c *gin.Context) {
	key := c.Param("key")
	var config models.SystemConfig
	if err := database.DB.Where("`key` = ?", key).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// UpdateConfig updates a system config value
func UpdateConfig(c *gin.Context) {
	key := c.Param("key")
	var body struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var config models.SystemConfig
	if err := database.DB.Where("`key` = ?", key).First(&config).Error; err != nil {
		// Create if not exists
		config = models.SystemConfig{Key: key, Value: body.Value}
		database.DB.Create(&config)
	} else {
		config.Value = body.Value
		database.DB.Save(&config)
	}

	c.JSON(http.StatusOK, config)
}

// getUserID extracts user ID from JWT claims in context
func getUserID(c *gin.Context) uint {
	if claims, exists := c.Get("currentUser"); exists {
		if cl, ok := claims.(*auth.Claims); ok {
			return cl.UserID
		}
	}
	return 0
}

func intFromDetails(details map[string]any, key string) int {
	if v, ok := details[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}
