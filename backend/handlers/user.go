package handlers

import (
	"dcmanager/auth"
	"dcmanager/database"
	"dcmanager/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func GetUsers(c *gin.Context) {
	var users []models.User
	database.DB.Preload("Role").Find(&users)
	c.JSON(http.StatusOK, users)
}

func GetUserOptions(c *gin.Context) {
	var users []models.User
	database.DB.Where("status = ?", "active").Order("display_name, username").Find(&users)
	options := make([]gin.H, 0, len(users))
	for _, user := range users {
		label := user.DisplayName
		if label == "" {
			label = user.Username
		}
		options = append(options, gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"display_name": user.DisplayName,
			"label":        label,
		})
	}
	c.JSON(http.StatusOK, options)
}

func CreateUser(c *gin.Context) {
	var body struct {
		Username    string `json:"username" binding:"required,min=3,max=50"`
		Password    string `json:"password" binding:"required,min=6"`
		DisplayName string `json:"display_name"`
		RoleID      uint   `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 检查用户名是否已存在
	var count int64
	database.DB.Model(&models.User{}).Where("username = ?", body.Username).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	// 检查角色是否存在
	var role models.Role
	if err := database.DB.First(&role, body.RoleID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色不存在"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	user := models.User{
		Username:     body.Username,
		PasswordHash: string(hash),
		DisplayName:  body.DisplayName,
		RoleID:       body.RoleID,
		Status:       "active",
	}
	database.DB.Create(&user)
	database.DB.Preload("Role").First(&user, user.ID)

	c.JSON(http.StatusCreated, user)
}

func UpdateUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	var body struct {
		DisplayName *string `json:"display_name"`
		RoleID      *uint   `json:"role_id"`
		Status      *string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if body.DisplayName != nil {
		user.DisplayName = *body.DisplayName
	}
	if body.RoleID != nil {
		var role models.Role
		if err := database.DB.First(&role, *body.RoleID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "角色不存在"})
			return
		}
		user.RoleID = *body.RoleID
	}
	if body.Status != nil {
		if *body.Status != "active" && *body.Status != "disabled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "状态值无效"})
			return
		}
		user.Status = *body.Status
	}

	database.DB.Save(&user)
	database.DB.Preload("Role").First(&user, user.ID)
	c.JSON(http.StatusOK, user)
}

func ResetPassword(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var body struct {
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码至少6个字符"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	user.PasswordHash = string(hash)
	database.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "密码重置成功"})
}

func DeleteUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	// 不允许删除自己
	claims := c.MustGet("currentUser").(*auth.Claims)
	if claims.UserID == uint(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除自己的账号"})
		return
	}

	result := database.DB.Delete(&models.User{}, id)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "用户已删除"})
}
