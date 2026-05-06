package config

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	CardHMACSecret    string
	CardPGPKey        string
	LogLevel          string
	SchedulerInterval time.Duration

	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	CBRMarginPercent float64
}

func Load() (Config, error) {
	_ = loadDotEnv(".env")

	interval, err := time.ParseDuration(env("SCHEDULER_INTERVAL", "12h"))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Port:              env("APP_PORT", "8080"),
		DatabaseURL:       env("DATABASE_URL", "postgres://bank:bank@localhost:5432/bank?sslmode=disable"),
		JWTSecret:         env("JWT_SECRET", ""),
		CardHMACSecret:    env("CARD_HMAC_SECRET", ""),
		CardPGPKey:        env("CARD_PGP_KEY", ""),
		LogLevel:          env("LOG_LEVEL", "info"),
		SchedulerInterval: interval,
		SMTPHost:          env("SMTP_HOST", ""),
		SMTPPort:          envInt("SMTP_PORT", 587),
		SMTPUser:          env("SMTP_USER", ""),
		SMTPPass:          env("SMTP_PASS", ""),
		SMTPFrom:          env("SMTP_FROM", "noreply@bank.local"),
		CBRMarginPercent:  envFloat("CBR_MARGIN_PERCENT", 5),
	}

	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	if len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must be at least 32 chars")
	}
	if len(c.CardHMACSecret) < 32 {
		return errors.New("CARD_HMAC_SECRET must be at least 32 chars")
	}
	if len(c.CardPGPKey) < 32 {
		return errors.New("CARD_PGP_KEY must be at least 32 chars")
	}
	return nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v, err := strconv.Atoi(env(key, ""))
	if err != nil {
		return fallback
	}
	return v
}

func envFloat(key string, fallback float64) float64 {
	v, err := strconv.ParseFloat(env(key, ""), 64)
	if err != nil {
		return fallback
	}
	return v
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)

		if key == "" {
			continue
		}

		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}

	return scanner.Err()
}
