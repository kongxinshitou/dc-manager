package handlers

import (
	"dcmanager/config"
	"dcmanager/database"
	"dcmanager/models"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadInspectionImages 上传巡检图片
func UploadInspectionImages(c *gin.Context) {
	inspectionID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的巡检ID"})
		return
	}

	// 检查巡检记录是否存在
	var inspection models.Inspection
	if err := database.DB.First(&inspection, inspectionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "巡检记录不存在"})
		return
	}

	// 检查已有图片数量
	var existCount int64
	database.DB.Model(&models.InspectionImage{}).Where("inspection_id = ?", inspectionID).Count(&existCount)

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法解析上传数据"})
		return
	}

	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择图片"})
		return
	}

	if int(existCount)+len(files) > config.MaxImagesPerInspection {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("每条巡检最多上传%d张图片（已有%d张，本次上传%d张）",
				config.MaxImagesPerInspection, existCount, len(files)),
		})
		return
	}

	// 确保目录存在
	dir := filepath.Join(config.UploadDir, "inspections", strconv.Itoa(inspectionID))
	os.MkdirAll(dir, 0755)

	var created []models.InspectionImage
	for _, file := range files {
		// 验证类型
		contentType := file.Header.Get("Content-Type")
		if !config.AllowedImageTypes[contentType] {
			continue // 跳过非图片文件
		}

		// 验证大小
		if file.Size > config.MaxUploadSize {
			continue // 跳过超大文件
		}

		// 生成唯一文件名
		ext := filepath.Ext(file.Filename)
		if ext == "" {
			ext = ".jpg"
		}
		filename := uuid.New().String() + ext
		relPath := filepath.Join("inspections", strconv.Itoa(inspectionID), filename)
		fullPath := filepath.Join(config.UploadDir, relPath)

		if err := c.SaveUploadedFile(file, fullPath); err != nil {
			continue
		}

		img := models.InspectionImage{
			InspectionID: uint(inspectionID),
			FilePath:     filepath.ToSlash(relPath),
			FileName:     file.Filename,
			FileSize:     file.Size,
			ContentType:  contentType,
		}
		database.DB.Create(&img)
		created = append(created, img)
	}

	c.JSON(http.StatusOK, created)
}

// GetInspectionImages 获取巡检图片列表
func GetInspectionImages(c *gin.Context) {
	inspectionID, _ := strconv.Atoi(c.Param("id"))
	var images []models.InspectionImage
	database.DB.Where("inspection_id = ?", inspectionID).Order("uploaded_at asc").Find(&images)
	c.JSON(http.StatusOK, images)
}

// DeleteInspectionImage 删除单张巡检图片
func DeleteInspectionImage(c *gin.Context) {
	inspectionID, _ := strconv.Atoi(c.Param("id"))
	imageID, _ := strconv.Atoi(c.Param("imageId"))

	var img models.InspectionImage
	if err := database.DB.Where("id = ? AND inspection_id = ?", imageID, inspectionID).First(&img).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "图片不存在"})
		return
	}

	// 删除物理文件
	fullPath := filepath.Join(config.UploadDir, img.FilePath)
	os.Remove(fullPath)

	// 删除数据库记录
	database.DB.Delete(&img)

	c.JSON(http.StatusOK, gin.H{"message": "图片已删除"})
}
