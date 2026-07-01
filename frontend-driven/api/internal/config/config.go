package config

import (
	"os"
)

type Config struct {
	DatabaseURL string
	ServerPort  string
	FrontendURL string

	CookieDomain string
	CookieSecure bool

	JWTSecret string

	GoogleClientID string

	MicrosoftClientID string
	MicrosoftTenant   string

	FacebookClientID     string
	FacebookClientSecret string

	GitHubClientID     string
	GitHubClientSecret string

	TelegramAPIToken string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://authuser:authpass@localhost:5432/authdb?sslmode=disable"),
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),

		CookieDomain: getEnv("COOKIE_DOMAIN", "localhost"),
		CookieSecure: getEnv("COOKIE_SECURE", "false") == "true",

		JWTSecret: getEnv("JWT_SECRET", ""),

		GoogleClientID: getEnv("GOOGLE_CLIENT_ID", ""),

		MicrosoftClientID: getEnv("MICROSOFT_CLIENT_ID", ""),
		MicrosoftTenant:   getEnv("MICROSOFT_TENANT", "common"),

		FacebookClientID:     getEnv("FACEBOOK_CLIENT_ID", ""),
		FacebookClientSecret: getEnv("FACEBOOK_CLIENT_SECRET", ""),

		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),

		TelegramAPIToken: getEnv("TELEGRAM_API_TOKEN", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
