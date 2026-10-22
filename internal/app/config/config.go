package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	ServiceHost string
	ServicePort int

	JWT   JWTConfig
	Redis RedisConfig
}

type JWTConfig struct {
	// Token — секрет, которым подписывается JWT.
	Token string
	// ExpiresIn — время жизни access-токена.
	ExpiresIn time.Duration
}

type RedisConfig struct {
	Host        string
	Port        int
	Password    string
	User        string
	DialTimeout time.Duration
	ReadTimeout time.Duration
}

func NewConfig() (*Config, error) {
	var err error

	configName := "config"
	_ = godotenv.Load()
	if os.Getenv("CONFIG_NAME") != "" {
		configName = os.Getenv("CONFIG_NAME")
	}

	viper.SetConfigName(configName)
	viper.SetConfigType("toml")
	viper.AddConfigPath("config")
	viper.AddConfigPath(".")
	viper.WatchConfig()

	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = viper.Unmarshal(cfg)
	if err != nil {
		return nil, err
	}

	applyEnvOverrides(cfg)

	if cfg.JWT.Token == "" {
		return nil, fmt.Errorf("JWT.Token не задан (config.toml или JWT_TOKEN)")
	}
	if cfg.JWT.ExpiresIn <= 0 {
		return nil, fmt.Errorf("JWT.ExpiresIn должен быть положительным")
	}
	if cfg.Redis.Host == "" || cfg.Redis.Port == 0 {
		return nil, fmt.Errorf("Redis.Host/Redis.Port не заданы (config.toml или REDIS_HOST/REDIS_PORT)")
	}

	log.Info("config parsed")

	return cfg, nil
}

// applyEnvOverrides позволяет хранить секреты вне репозитория:
// значения из окружения имеют приоритет над config.toml.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("JWT_TOKEN"); v != "" {
		cfg.JWT.Token = v
	}
	if v := os.Getenv("JWT_EXPIRES_IN"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.JWT.ExpiresIn = d
		}
	}

	if v := os.Getenv("REDIS_HOST"); v != "" {
		cfg.Redis.Host = v
	}
	if v := os.Getenv("REDIS_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Redis.Port = p
		}
	}
	if v, ok := os.LookupEnv("REDIS_PASSWORD"); ok {
		cfg.Redis.Password = v
	}
	if v, ok := os.LookupEnv("REDIS_USER"); ok {
		cfg.Redis.User = v
	}
}
