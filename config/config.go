package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerAddr  string
	DatabaseURL string
	JWTSecret   string
	EmailFrom   string
	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	SMTPPass    string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "4130"
	}
	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost"
	}

	smtpPortStr := os.Getenv("SMTP_PORT")
	if smtpPortStr == "" {
		return nil, fmt.Errorf("SMTP_PORT is not set in the environment")
	}
	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}

	cfg := &Config{
		ServerAddr:  fmt.Sprintf("%s:%s", host, port),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		EmailFrom:   os.Getenv("EMAIL_FROM"),
		SMTPHost:    os.Getenv("SMTP_HOST"),
		SMTPPort:    smtpPort,
		SMTPUser:    os.Getenv("SMTP_USER"),
		SMTPPass:    os.Getenv("SMTP_PASS"),
	}

	if cfg.DatabaseURL == "" || cfg.JWTSecret == "" {
		return nil, fmt.Errorf("DATABASE_URL and JWT_SECRET must be set")
	}

	return cfg, nil
}
