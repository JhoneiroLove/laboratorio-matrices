package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/interseguro/matrix-api/internal/domain"
)

type Config struct {
	Address          string
	BodyLimit        int
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	IdleTimeout      time.Duration
	ShutdownTimeout  time.Duration
	AuthRateLimitMax int
	AuthRateWindow   time.Duration
	Limits           domain.Limits
	JWT              JWT
	Demo             DemoCredentials
	Analytics        Analytics
}

type JWT struct {
	PrivateKeyPath string
	PublicKeyPath  string
	Issuer         string
	Audience       string
	TTL            time.Duration
	Leeway         time.Duration
}

type DemoCredentials struct {
	Username string
	Password string
}

type Analytics struct {
	BaseURL string
	Timeout time.Duration
}

func Load() (Config, error) {
	config := Config{
		Address:          env("MATRIX_API_ADDR", ":8080"),
		BodyLimit:        envInt("HTTP_BODY_LIMIT_BYTES", 1<<20),
		ReadTimeout:      envDuration("HTTP_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:     envDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:      envDuration("HTTP_IDLE_TIMEOUT", 30*time.Second),
		ShutdownTimeout:  envDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		AuthRateLimitMax: envInt("AUTH_RATE_LIMIT_MAX", 5),
		AuthRateWindow:   envDuration("AUTH_RATE_LIMIT_WINDOW", time.Minute),
		Limits: domain.Limits{
			MaxRows:     envInt("MATRIX_MAX_ROWS", 256),
			MaxColumns:  envInt("MATRIX_MAX_COLUMNS", 256),
			MaxElements: envInt("MATRIX_MAX_ELEMENTS", 65_536),
		},
		JWT: JWT{
			PrivateKeyPath: os.Getenv("JWT_PRIVATE_KEY_PATH"),
			PublicKeyPath:  os.Getenv("JWT_PUBLIC_KEY_PATH"),
			Issuer:         env("JWT_ISSUER", "matrix-api"),
			Audience:       env("JWT_AUDIENCE", "matrix-api"),
			TTL:            envDuration("JWT_TTL", 15*time.Minute),
			Leeway:         envDuration("JWT_LEEWAY", 30*time.Second),
		},
		Demo: DemoCredentials{
			Username: os.Getenv("DEMO_USERNAME"),
			Password: os.Getenv("DEMO_PASSWORD"),
		},
		Analytics: Analytics{
			BaseURL: env("ANALYTICS_BASE_URL", "http://analytics-api:3000"),
			Timeout: envDuration("ANALYTICS_TIMEOUT", 5*time.Second),
		},
	}

	if config.BodyLimit <= 0 || config.ReadTimeout <= 0 || config.WriteTimeout <= 0 || config.IdleTimeout <= 0 || config.ShutdownTimeout <= 0 || config.AuthRateLimitMax <= 0 || config.AuthRateWindow <= 0 {
		return Config{}, fmt.Errorf("los límites y tiempos de espera HTTP deben ser positivos")
	}
	if config.Limits.MaxRows <= 0 || config.Limits.MaxColumns <= 0 || config.Limits.MaxElements <= 0 {
		return Config{}, fmt.Errorf("los límites de las matrices deben ser positivos")
	}
	if config.JWT.PrivateKeyPath == "" || config.JWT.PublicKeyPath == "" {
		return Config{}, fmt.Errorf("JWT_PRIVATE_KEY_PATH y JWT_PUBLIC_KEY_PATH son obligatorios")
	}
	if config.JWT.Issuer == "" || config.JWT.Audience == "" || config.JWT.TTL < time.Second || config.JWT.Leeway < 0 {
		return Config{}, fmt.Errorf("el issuer, el audience y las duraciones válidas del JWT son obligatorios")
	}
	if config.Demo.Username == "" || config.Demo.Password == "" {
		return Config{}, fmt.Errorf("DEMO_USERNAME y DEMO_PASSWORD son obligatorios")
	}
	if config.Analytics.BaseURL == "" || config.Analytics.Timeout <= 0 {
		return Config{}, fmt.Errorf("la URL y el tiempo de espera del servicio de estadísticas son obligatorios")
	}
	return config, nil
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	return parsed
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return -1
	}
	return parsed
}
