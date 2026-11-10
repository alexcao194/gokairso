package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DB      DBConfig
	App     AppConfig
	Redis   RedisConfig
	Storage StorageConfig
}
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	MaxConns string
}
type RedisConfig struct {
	Port    string
	Address string
}
type StorageConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UseSSL          bool
	PublicURL       string
}
type AppConfig struct {
	JWTSecret               string
	AccessTokenDuration     time.Duration
	RefreshTokenDuration    time.Duration
	FirebaseCredentialsPath string
	AppTokenSecret          string
}

func (db DBConfig) DatabaseURL() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s&pool_max_conns=%s",
		url.QueryEscape(db.User), url.QueryEscape(db.Password), db.Host, db.Port, db.Name, db.SSLMode,
		url.QueryEscape(db.MaxConns),
	)
}
func Load() Config {
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found, reading from environment")
	}

	return Config{
		DB: DBConfig{
			Host:     mustEnv("DB_HOST"),
			Port:     mustEnv("DB_PORT"),
			User:     mustEnv("DB_USER"),
			Password: mustEnv("DB_PASSWORD"),
			Name:     mustEnv("DB_NAME"),
			SSLMode:  mustEnv("DB_SSLMODE"),
			MaxConns: envDefault("DB_MAX_CONNS", "20"),
		},
		App: AppConfig{
			JWTSecret:               mustEnv("JWT_SECRET"),
			AccessTokenDuration:     parseDuration(mustEnv("JWT_ACCESS_EXPIRATION")),
			RefreshTokenDuration:    parseDuration(mustEnv("JWT_REFRESH_EXPIRATION")),
			FirebaseCredentialsPath: mustEnv("FIREBASE_CREDENTIALS_PATH"),
			AppTokenSecret:          mustEnv("APP_TOKEN_SECRET"),
		},
		Redis: RedisConfig{
			Port:    mustEnv("REDIS_PORT"),
			Address: fmt.Sprintf("%s:%s", mustEnv("REDIS_HOST"), mustEnv("REDIS_PORT")),
		},
		Storage: StorageConfig{
			Endpoint:        mustEnv("MINIO_ENDPOINT"),
			AccessKeyID:     mustEnv("MINIO_ACCESS_KEY"),
			SecretAccessKey: mustEnv("MINIO_SECRET_KEY"),
			Bucket:          mustEnv("MINIO_BUCKET"),
			UseSSL:          parseBool(envDefault("MINIO_USE_SSL", "false")),
			PublicURL:       envDefault("MINIO_PUBLIC_URL", ""),
		},
	}
}

func parseBool(s string) bool {
	return s == "true" || s == "1"
}
func envDefault(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return value
}
func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		slog.Error("Invalid duration format", "duration", s, "error", err)
		os.Exit(1)
	}
	return d
}
func mustEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		slog.Error("Required environment variable is not set", "key", key)
		os.Exit(1)
	}
	return value
}
