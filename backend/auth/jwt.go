package auth

import (
	"dcmanager/models"
	"encoding/json"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(getEnvOrDefault("JWT_SECRET", "dcmanager-default-secret-change-me"))

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

const TokenExpiration = 24 * time.Hour

type Claims struct {
	UserID      uint     `json:"user_id"`
	Username    string   `json:"username"`
	RoleID      uint     `json:"role_id"`
	RoleName    string   `json:"role_name"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// GenerateToken 为用户生成 JWT token
func GenerateToken(user *models.User, role *models.Role) (string, error) {
	var perms []string
	if role != nil && role.Permissions != "" {
		json.Unmarshal([]byte(role.Permissions), &perms)
	}
	claims := Claims{
		UserID:      user.ID,
		Username:    user.Username,
		RoleID:      user.RoleID,
		RoleName:    role.Name,
		Permissions: perms,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken 解析并验证 JWT token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrSignatureInvalid
}
