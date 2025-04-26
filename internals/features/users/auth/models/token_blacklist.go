package models

import (
	"time"
	"gorm.io/gorm"
)

type TokenBlacklist struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Token     string         `gorm:"type:text;not null;unique" json:"token"`
	ExpiredAt time.Time      `json:"expired_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName memastikan nama tabel sesuai dengan skema database
func (TokenBlacklist) TableName() string {
	return "token_blacklist"
}
