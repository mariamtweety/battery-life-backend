package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"battery-life/internal/app/config"
	"battery-life/internal/app/ds"
	"battery-life/internal/app/redisclient"
	"battery-life/internal/app/role"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// Issuer — обязательный издатель токена, проверяется при парсинге.
	Issuer = "battery-life"
	// AccessTokenCookie — имя cookie, в которую дублируется JWT (удобно для Swagger/SPA).
	AccessTokenCookie = "access_token"
	// BearerPrefix — префикс значения заголовка Authorization.
	BearerPrefix = "Bearer "
)

var (
	ErrNoToken      = errors.New("токен не передан")
	ErrInvalidToken = errors.New("токен недействителен")
	ErrBlacklisted  = errors.New("токен отозван (находится в blacklist)")
)

// Manager выпускает и проверяет JWT, а также хранит сессии/blacklist в Redis.
type Manager struct {
	cfg   config.JWTConfig
	redis *redisclient.Client
}

func New(cfg config.JWTConfig, redis *redisclient.Client) *Manager {
	return &Manager{cfg: cfg, redis: redis}
}

// Generate выпускает подписанный access-токен для пользователя.
func (m *Manager) Generate(user *ds.BatteryAppUser) (string, *ds.JWTClaims, error) {
	now := time.Now()
	claims := &ds.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    Issuer,
			Subject:   user.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.cfg.ExpiresIn)),
		},
		UserID:   user.ID,
		Username: user.Username,
		Role:     role.FromModeratorFlag(user.IsModerator),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(m.cfg.Token))
	if err != nil {
		return "", nil, fmt.Errorf("не удалось подписать JWT: %w", err)
	}

	return signed, claims, nil
}

// Parse проверяет подпись, срок действия и издателя токена.
func (m *Manager) Parse(tokenStr string) (*ds.JWTClaims, error) {
	claims := &ds.JWTClaims{}

	parsed, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
			}
			return []byte(m.cfg.Token), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !parsed.Valid {
		return nil, ErrInvalidToken
	}
	if !claims.Role.IsValid() {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// Authenticate проверяет blacklist и валидность токена.
func (m *Manager) Authenticate(ctx context.Context, tokenStr string) (*ds.JWTClaims, error) {
	if tokenStr == "" {
		return nil, ErrNoToken
	}

	blacklisted, err := m.redis.CheckJWTInBlacklist(ctx, tokenStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки blacklist в Redis: %w", err)
	}
	if blacklisted {
		return nil, ErrBlacklisted
	}

	return m.Parse(tokenStr)
}

// SaveSession сохраняет сессию в Redis — по ней в redis-cli видно,
// какому пользователю был выдан токен (требование демонстрации ЛР4).
func (m *Manager) SaveSession(ctx context.Context, claims *ds.JWTClaims) error {
	ttl := time.Until(claims.ExpiresAtTime())
	if ttl <= 0 {
		ttl = m.cfg.ExpiresIn
	}

	return m.redis.SaveSession(ctx, claims.ID, redisclient.SessionData{
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     int(claims.Role),
		RoleName: claims.Role.String(),
		IssuedAt: time.Now(),
	}, ttl)
}

// Logout добавляет токен в blacklist и удаляет запись сессии.
func (m *Manager) Logout(ctx context.Context, tokenStr string, claims *ds.JWTClaims) error {
	ttl := time.Until(claims.ExpiresAtTime())
	if ttl <= 0 {
		// Токен уже истёк — в blacklist его хранить не нужно.
		return m.redis.DeleteSession(ctx, claims.ID)
	}

	if err := m.redis.WriteJWTToBlacklist(ctx, tokenStr, ttl); err != nil {
		return fmt.Errorf("не удалось добавить токен в blacklist: %w", err)
	}
	return m.redis.DeleteSession(ctx, claims.ID)
}
