package models

import "time"

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Username     string    `json:"username" gorm:"uniqueIndex;size:50;not null"`
	PasswordHash string    `json:"-" gorm:"not null"`
	DisplayName  string    `json:"display_name" gorm:"size:100"`
	RoleID       uint      `json:"role_id" gorm:"not null;default:1"`
	Role         *Role     `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	Status       string    `json:"status" gorm:"size:20;not null;default:'active'"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
