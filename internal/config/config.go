package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPAddr       string
	PostgresDSN    string
	SQSQueueURL    string
	SQSQueueURLDLQ string
	OAuthIssuerURL string
	OAuthAudience  string
	OAuthClientID  string
	UseStubAuth    bool
}

func Load() Config {
	return Config{
		HTTPAddr:       getenv("HTTP_ADDR", ":8080"),
		PostgresDSN:    getenv("POSTGRES_DSN", "memory"),
		SQSQueueURL:    getenv("SQS_QUEUE_URL", ""),
		SQSQueueURLDLQ: getenv("SQS_QUEUE_URL_DLQ", ""),
		OAuthIssuerURL: getenv("OIDC_ISSUER_URL", ""),
		OAuthAudience:  getenv("OIDC_AUDIENCE", "desafio-backend"),
		OAuthClientID:  getenv("OIDC_CLIENT_ID", "desafio-backend"),
		UseStubAuth:    getenvBool("USE_STUB_AUTH", true),
	}
}

func (c Config) Validate() error {
	if c.HTTPAddr == "" {
		return fmt.Errorf("HTTP_ADDR is required")
	}
	return nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if value == "true" || value == "1" || value == "yes" {
			return true
		}
		return false
	}
	return fallback
}
