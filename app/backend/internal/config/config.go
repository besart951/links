package config

import "os"

type Config struct {
	Port        string
	PublicURL   string
	Env         string
	DatabaseURL string
}

func Load() Config {
	return Config{
		Port:        getEnv("PORT_BACKEND", "4000"),
		PublicURL:   getEnv("PUBLIC_API_URL", "http://localhost:4000"),
		Env:         getEnv("APP_ENV", "development"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
