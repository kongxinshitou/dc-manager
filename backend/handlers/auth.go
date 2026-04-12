package handlers

import (
	"dcmanager/auth"
	"dcmanager/database"
	"dcmanager/models"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func Login(c *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入用户名和密码"})
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", body.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已被禁用"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	var role models.Role
	database.DB.First(&role, user.RoleID)

	token, err := auth.GenerateToken(&user, &role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"display_name": user.DisplayName,
			"role_id":      user.RoleID,
			"role_name":    role.Name,
			"permissions":  getPermissionsFromRole(&role),
		},
	})
}

func ChangePassword(c *gin.Context) {
	claims := c.MustGet("currentUser").(*auth.Claims)

	var body struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码至少6个字符"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, claims.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.OldPassword)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "原密码错误"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	user.PasswordHash = string(hash)
	database.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}

func getPermissionsFromRole(role *models.Role) []string {
	if role == nil || role.Permissions == "" {
		return nil
	}
	if role.Name == "admin" {
		return models.AllPermissions
	}
	var perms []string
	_ = json.Unmarshal([]byte(role.Permissions), &perms)
	return perms
}
