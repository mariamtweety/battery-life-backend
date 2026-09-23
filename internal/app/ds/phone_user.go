package ds

import "time"

type PhoneUser struct {
	ID           uint      `gorm:"primaryKey;column:id" json:"id"`
	Username     string    `gorm:"type:varchar(50);unique;not null;column:username" json:"username"`
	PasswordHash string    `gorm:"type:varchar(100);not null;column:password_hash" json:"-"`
	IsModerator  bool      `gorm:"type:boolean;not null;default:false;column:is_moderator" json:"is_moderator"`
	CreatedAt    time.Time `gorm:"not null;column:created_at" json:"created_at"`
}

func (PhoneUser) TableName() string {
	return "phone_users"
}
