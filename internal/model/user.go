package model

import "time"

type Role struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"uniqueIndex;size:50;not null"`
	Permissions JSON      `json:"permissions" gorm:"type:json;not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type User struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	Username     string     `json:"username" gorm:"uniqueIndex;size:100;not null"`
	Email        string     `json:"email" gorm:"uniqueIndex;size:255;not null"`
	PasswordHash string     `json:"-" gorm:"size:255;not null"`
	RoleID       uint       `json:"role_id" gorm:"not null"`
	Role         *Role      `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	IsActive     bool       `json:"is_active" gorm:"default:true;not null"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type RefreshToken struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	TokenHash string    `json:"-" gorm:"uniqueIndex;size:255;not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"index;not null"`
	CreatedAt time.Time `json:"created_at"`
}
