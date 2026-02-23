package models

import (
	"time"
)

type User struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name string `gorm:"not null" json:"name"`
	Email string `gorm:"uniqueIndex;not null" json:"email"`
	Password string `gorm:"not null" json:"password"`
	ResetCode *string `json:"resetCode"`
	ResetCodeExpires *time.Time `json:"resetCodeExpires"`
	Created time.Time `gorm:"not null" json:"created"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

