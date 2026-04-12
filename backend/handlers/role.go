package handlers

import (
	"dcmanager/database"
	"dcmanager/models"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetRoles(c *gin.Context) {
	var roles []models.Role
	database.DB.Find(&roles)
	c.JSON(http.StatusOK, roles)
}

func CreateRole(c *gin.Context) {
	var body struct {
		Name        string   `json:"name" binding:"required,min=1,max=50"`
		DisplayName string   `json:"display_name" binding:"required"`
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 验证权限码合法性
	if err := validatePermissions(body.Permissions); err != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	permsJSON, _ := json.Marshal(body.Permissions)
	role := models.Role{
		Name:        body.Name,
		DisplayName: body.DisplayName,
		Permissions: string(permsJSON),
		IsSystem:    false,
	}

	if err := database.DB.Create(&role).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "角色名称已存在"})
		return
	}

	c.JSON(http.StatusCreated, role)
}

func UpdateRole(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var role models.Role
	if err := database.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": "系统内置角色不可修改"})
		return
	}

	var body struct {
		Name        *string  `json:"name"`
		DisplayName *string  `json:"display_name"`
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if body.Name != nil {
		role.Name = *body.Name
	}
	if body.DisplayName != nil {
		role.DisplayName = *body.DisplayName
	}
	if body.Permissions != nil {
		if err := validatePermissions(body.Permissions); err != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			return
		}
		permsJSON, _ := json.Marshal(body.Permissions)
		role.Permissions = string(permsJSON)
	}

	database.DB.Save(&role)
	c.JSON(http.StatusOK, role)
}

func DeleteRole(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var role models.Role
	if err := database.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": "系统内置角色不可删除"})
		return
	}

	// 检查是否有用户使用此角色
	var count int64
	database.DB.Model(&models.User{}).Where("role_id = ?", role.ID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该角色下有用户，无法删除"})
		return
	}

	database.DB.Delete(&role)
	c.JSON(http.StatusOK, gin.H{"message": "角色已删除"})
}

// GetPermissionInfo 返回权限码定义，供前端渲染配置页面
func GetPermissionInfo(c *gin.Context) {
	type permItem struct {
		Code  string `json:"code"`
		Label string `json:"label"`
	}
	type permGroup struct {
		Label       string     `json:"label"`
		Permissions []permItem `json:"permissions"`
	}

	groups := make([]permGroup, len(models.PermissionGroups))
	for i, g := range models.PermissionGroups {
		items := make([]permItem, len(g.Permissions))
		for j, p := range g.Permissions {
			items[j] = permItem{Code: p, Label: models.PermissionLabels[p]}
		}
		groups[i] = permGroup{Label: g.Label, Permissions: items}
	}

	c.JSON(http.StatusOK, gin.H{
		"groups": groups,
		"all":    models.AllPermissions,
	})
}

func validatePermissions(perms []string) string {
	validSet := make(map[string]bool, len(models.AllPermissions))
	for _, p := range models.AllPermissions {
		validSet[p] = true
	}
	for _, p := range perms {
		if !validSet[p] {
			return "无效的权限码: " + p
		}
	}
	return ""
}
