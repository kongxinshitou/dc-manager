package middleware

import (
	"dcmanager/auth"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthRequired 验证 JWT token 并将 Claims 存入上下文
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "缺少认证信息"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "认证格式错误"})
			return
		}

		claims, err := auth.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的认证信息"})
			return
		}

		c.Set("currentUser", claims)
		c.Next()
	}
}

// PermissionRequired 检查当前用户是否拥有指定权限之一
func PermissionRequired(requiredPerms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("currentUser")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			return
		}

		userClaims := claims.(*auth.Claims)

		// admin 角色直接放行
		if userClaims.RoleName == "admin" {
			c.Next()
			return
		}

		permSet := make(map[string]bool, len(userClaims.Permissions))
		for _, p := range userClaims.Permissions {
			permSet[p] = true
		}

		for _, rp := range requiredPerms {
			if permSet[rp] {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限不足"})
	}
}
