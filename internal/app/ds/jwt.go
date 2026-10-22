package ds

import (
	"time"

	"battery-life/internal/app/role"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims — полезная нагрузка access-токена.
// Поля UserID/Role вычисляются на бэкенде и с клиента не принимаются.
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID   uint      `json:"user_id"`
	Username string    `json:"username"`
	Role     role.Role `json:"role"`
}

// ExpiresAtTime возвращает время истечения токена.
func (c *JWTClaims) ExpiresAtTime() time.Time {
	if c.ExpiresAt == nil {
		return time.Time{}
	}
	return c.ExpiresAt.Time
}
