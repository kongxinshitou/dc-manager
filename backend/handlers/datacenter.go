package handlers

import (
	"dcmanager/database"
	"dcmanager/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ========== Datacenter CRUD ==========

func GetDatacenters(c *gin.Context) {
	var datacenters []models.Datacenter
	database.DB.Order("id").Find(&datacenters)
	c.JSON(http.StatusOK, datacenters)
}

func CreateDatacenter(c *gin.Context) {
	var dc models.Datacenter
	if err := c.ShouldBindJSON(&dc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dc.ID = 0
	if err := database.DB.Create(&dc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, dc)
}

func UpdateDatacenter(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var dc models.Datacenter
	if err := database.DB.First(&dc, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := c.ShouldBindJSON(&dc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dc.ID = uint(id)
	database.DB.Save(&dc)
	c.JSON(http.StatusOK, dc)
}

func DeleteDatacenter(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	// Check for linked cabinets
	var count int64
	database.DB.Model(&models.Cabinet{}).Where("datacenter_id = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该机房下还有机柜，无法删除"})
		return
	}
	database.DB.Delete(&models.Datacenter{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ========== Column Management ==========

func GetDatacenterColumns(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var columns []models.CabinetColumn
	database.DB.Where("datacenter_id = ?", id).Order("sort_order, id").Find(&columns)
	c.JSON(http.StatusOK, columns)
}

func SetDatacenterColumns(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body struct {
		Columns []models.CabinetColumn `json:"columns" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Delete existing and recreate
	database.DB.Where("datacenter_id = ?", id).Delete(&models.CabinetColumn{})
	for i := range body.Columns {
		body.Columns[i].ID = 0
		body.Columns[i].DatacenterID = uint(id)
		if body.Columns[i].SortOrder == 0 {
			body.Columns[i].SortOrder = i
		}
		if body.Columns[i].ColumnType == "" {
			body.Columns[i].ColumnType = "cabinet"
		}
	}
	database.DB.Create(&body.Columns)
	c.JSON(http.StatusOK, body.Columns)
}

// ========== Row Management ==========

func GetDatacenterRows(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var rows []models.CabinetRow
	database.DB.Where("datacenter_id = ?", id).Order("sort_order, id").Find(&rows)
	c.JSON(http.StatusOK, rows)
}

func SetDatacenterRows(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body struct {
		Rows []models.CabinetRow `json:"rows" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database.DB.Where("datacenter_id = ?", id).Delete(&models.CabinetRow{})
	for i := range body.Rows {
		body.Rows[i].ID = 0
		body.Rows[i].DatacenterID = uint(id)
		if body.Rows[i].SortOrder == 0 {
			body.Rows[i].SortOrder = i
		}
	}
	database.DB.Create(&body.Rows)
	c.JSON(http.StatusOK, body.Rows)
}

// ========== Cabinet Management ==========

func GetDatacenterCabinets(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cabinets []models.Cabinet
	database.DB.Where("datacenter_id = ?", id).Order("id").Find(&cabinets)
	c.JSON(http.StatusOK, cabinets)
}

func GenerateCabinets(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var dc models.Datacenter
	if err := database.DB.First(&dc, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "datacenter not found"})
		return
	}

	var body struct {
		DefaultHeight int `json:"default_height"`
	}
	c.ShouldBindJSON(&body)
	if body.DefaultHeight <= 0 {
		body.DefaultHeight = dc.MaxU
		if body.DefaultHeight <= 0 {
			body.DefaultHeight = 47
		}
	}

	// Get columns and rows
	var columns []models.CabinetColumn
	database.DB.Where("datacenter_id = ?", id).Order("sort_order, id").Find(&columns)

	var rows []models.CabinetRow
	database.DB.Where("datacenter_id = ?", id).Order("sort_order, id").Find(&rows)

	if len(columns) == 0 || len(rows) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置列和行"})
		return
	}

	// Delete existing cabinets
	database.DB.Where("datacenter_id = ?", id).Delete(&models.Cabinet{})

	// Generate cabinets from columns x rows
	var cabinets []models.Cabinet
	for _, row := range rows {
		for _, col := range columns {
			cabinetType := "standard"
			if col.ColumnType != "cabinet" && col.ColumnType != "" {
				cabinetType = col.ColumnType
			}
			height := body.DefaultHeight
			cabinets = append(cabinets, models.Cabinet{
				DatacenterID: uint(id),
				ColumnID:     &col.ID,
				RowID:        &row.ID,
				Name:         col.Name + "-" + row.Name,
				Height:       height,
				Width:        60,
				Depth:        120,
				CabinetType:  cabinetType,
			})
		}
	}

	if len(cabinets) > 0 {
		database.DB.Create(&cabinets)
	}

	c.JSON(http.StatusOK, gin.H{
		"generated": len(cabinets),
		"cabinets":  cabinets,
	})
}

func UpdateCabinet(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cabinet models.Cabinet
	if err := database.DB.First(&cabinet, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := c.ShouldBindJSON(&cabinet); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cabinet.ID = uint(id)
	database.DB.Save(&cabinet)
	c.JSON(http.StatusOK, cabinet)
}

func GetCabinetDevices(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cabinet models.Cabinet
	if err := database.DB.First(&cabinet, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cabinet not found"})
		return
	}

	var devices []models.Device
	database.DB.Where("cabinet_id = ?", id).Order("start_u").Find(&devices)

	c.JSON(http.StatusOK, gin.H{
		"cabinet": cabinet,
		"devices": devices,
	})
}

// ========== Layout API ==========

func GetDatacenterLayout(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var dc models.Datacenter
	if err := database.DB.First(&dc, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "datacenter not found"})
		return
	}

	var columns []models.CabinetColumn
	database.DB.Where("datacenter_id = ?", id).Order("sort_order, id").Find(&columns)

	var rows []models.CabinetRow
	database.DB.Where("datacenter_id = ?", id).Order("sort_order, id").Find(&rows)

	var cabinets []models.Cabinet
	database.DB.Where("datacenter_id = ?", id).Order("id").Find(&cabinets)

	// Get devices for all cabinets in this datacenter
	var devices []models.Device
	database.DB.Where("datacenter_id = ?", id).Find(&devices)

	// Build device map by cabinet_id
	devicesByCabinet := make(map[uint][]map[string]any)
	for _, d := range devices {
		entry := map[string]any{
			"id":         d.ID,
			"brand":      d.Brand,
			"model":      d.Model,
			"device_type": d.DeviceType,
			"start_u":    d.StartU,
			"end_u":      d.EndU,
			"status":     d.DeviceStatus,
			"sub_status":  d.SubStatus,
		}
		if d.CabinetID != nil {
			devicesByCabinet[*d.CabinetID] = append(devicesByCabinet[*d.CabinetID], entry)
		}
	}

	// Build cabinet layout with devices
	type CabinetLayout struct {
		models.Cabinet
		Devices []map[string]any `json:"devices"`
		UsedU   int              `json:"used_u"`
	}

	var cabinetLayouts []CabinetLayout
	for _, cab := range cabinets {
		devs := devicesByCabinet[cab.ID]
		usedU := 0
		for _, d := range devs {
			if su, ok := d["start_u"].(*int); ok && su != nil {
				if eu, ok := d["end_u"].(*int); ok && eu != nil {
					usedU += *eu - *su + 1
				}
			}
		}
		cabinetLayouts = append(cabinetLayouts, CabinetLayout{
			Cabinet: cab,
			Devices: devs,
			UsedU:   usedU,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"datacenter": dc,
		"columns":    columns,
		"rows":       rows,
		"cabinets":   cabinetLayouts,
	})
}
