package redisclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"battery-life/internal/app/config"

	goredis "github.com/redis/go-redis/v9"
)

// servicePrefix отделяет ключи сервиса от ключей других приложений в общем Redis.
const servicePrefix = "battery_life."

type Client struct {
	client *goredis.Client
}

// New создаёт клиента и проверяет соединение через PING.
func New(ctx context.Context, cfg config.RedisConfig) (*Client, error) {
	c := goredis.NewClient(&goredis.Options{
		Addr:        cfg.Host + ":" + strconv.Itoa(cfg.Port),
		Username:    cfg.User,
		Password:    cfg.Password,
		DB:          0,
		DialTimeout: cfg.DialTimeout,
		ReadTimeout: cfg.ReadTimeout,
	})

	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("не удалось подключиться к Redis (%s:%d): %w", cfg.Host, cfg.Port, err)
	}

	return &Client{client: c}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Underlying даёт доступ к клиенту для служебных сценариев.
func (c *Client) Underlying() *goredis.Client {
	return c.client
}

// SessionData — то, что видно в Redis по ключу сессии (удобно для демонстрации ЛР4).
type SessionData struct {
	UserID   uint      `json:"user_id"`
	Username string    `json:"username"`
	Role     int       `json:"role"`
	RoleName string    `json:"role_name"`
	IssuedAt time.Time `json:"issued_at"`
}

func sessionKey(jti string) string {
	return servicePrefix + "session." + jti
}

func blacklistKey(token string) string {
	return servicePrefix + "jwt.blacklist." + token
}

// SaveSession сохраняет запись о выданном токене до момента его истечения.
func (c *Client) SaveSession(ctx context.Context, jti string, data SessionData, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = time.Hour
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("не удалось сериализовать сессию: %w", err)
	}
	return c.client.Set(ctx, sessionKey(jti), payload, ttl).Err()
}

// DeleteSession удаляет запись сессии (вызывается при logout).
func (c *Client) DeleteSession(ctx context.Context, jti string) error {
	return c.client.Del(ctx, sessionKey(jti)).Err()
}

// WriteJWTToBlacklist добавляет токен в blacklist на оставшееся время его жизни.
func (c *Client) WriteJWTToBlacklist(ctx context.Context, token string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = time.Minute
	}
	return c.client.Set(ctx, blacklistKey(token), "1", ttl).Err()
}

// CheckJWTInBlacklist возвращает true, если токен уже разлогинен.
func (c *Client) CheckJWTInBlacklist(ctx context.Context, token string) (bool, error) {
	err := c.client.Get(ctx, blacklistKey(token)).Err()
	if err == goredis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
