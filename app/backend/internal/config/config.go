package config

import "os"

type Config struct {
	Port                     string
	PublicURL                string
	Env                      string
	DatabaseURL              string
	SessionCookieName        string
	SessionCookieSecure      bool
	BootstrapSuperAdminEmail string
	AllowedOrigins           string
}

func Load() Config {
	env := getEnv("APP_ENV", "development")
	return Config{
		Port:                     getEnv("PORT_BACKEND", "4000"),
		PublicURL:                getEnv("PUBLIC_API_URL", "http://localhost:4000"),
		Env:                      env,
		DatabaseURL:              getEnv("DATABASE_URL", ""),
		SessionCookieName:        getEnv("SESSION_COOKIE_NAME", "links_session"),
		SessionCookieSecure:      getEnvBool("SESSION_COOKIE_SECURE", env != "development"),
		BootstrapSuperAdminEmail: getEnv("BOOTSTRAP_SUPER_ADMIN_EMAIL", ""),
		AllowedOrigins:           getEnv("CORS_ALLOWED_ORIGINS", "*"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v == "1" || v == "true" || v == "TRUE" || v == "yes" || v == "YES"
}
