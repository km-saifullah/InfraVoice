package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	App       AppConfig
	Server    ServerConfig
	MongoDB   MongoDBConfig
	Redis     RedisConfig
	JWT       JWTConfig
	AI        AIConfig
	Terraform TerraformConfig
}

type AppConfig struct {
	Name string
	Env  string
}

type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeoutSec  int
	WriteTimeoutSec int
	IdleTimeoutSec  int
}

type MongoDBConfig struct {
	URI      string
	Database string
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret             string
	AccessTokenMinutes int
}

type AIConfig struct {
	Provider string
	APIKey   string
	Model    string
}

type TerraformConfig struct {
	BinaryPath          string
	WorkspaceRoot       string
	ExecutionTimeoutSec int
}

func Load() (Config, error) {
	cfg := Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "infra-voice-api"),
			Env:  getEnv("APP_ENV", "development"),
		},

		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnvInt("SERVER_PORT", 8000),
			ReadTimeoutSec:  getEnvInt("SERVER_READ_TIMEOUT_SEC", 15),
			WriteTimeoutSec: getEnvInt("SERVER_WRITE_TIMEOUT_SEC", 15),
			IdleTimeoutSec:  getEnvInt("SERVER_IDLE_TIMEOUT_SEC", 60),
		},

		MongoDB: MongoDBConfig{
			URI:      getEnv("MONGODB_URI", ""),
			Database: getEnv("MONGODB_DATABASE", "infra_voice"),
		},

		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},

		JWT: JWTConfig{
			Secret:             getEnv("JWT_SECRET", ""),
			AccessTokenMinutes: getEnvInt("JWT_ACCESS_TOKEN_MINUTES", 15),
		},

		AI: AIConfig{
			Provider: getEnv("AI_PROVIDER", "openai"),
			APIKey:   getEnv("AI_API_KEY", ""),
			Model:    getEnv("AI_MODEL", ""),
		},

		Terraform: TerraformConfig{
			BinaryPath:          getEnv("TERRAFORM_BINARY_PATH", "terraform"),
			WorkspaceRoot:       getEnv("TERRAFORM_WORKSPACE_ROOT", "./terraform/workspaces"),
			ExecutionTimeoutSec: getEnvInt("TERRAFORM_EXECUTION_TIMEOUT_SEC", 300),
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	var validationErrors []string

	if strings.TrimSpace(c.App.Name) == "" {
		validationErrors = append(
			validationErrors,
			"APP_NAME cannot be empty",
		)
	}

	if strings.TrimSpace(c.App.Env) == "" {
		validationErrors = append(
			validationErrors,
			"APP_ENV cannot be empty",
		)
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		validationErrors = append(
			validationErrors,
			"SERVER_PORT must be between 1 and 65535",
		)
	}

	if c.Server.ReadTimeoutSec <= 0 {
		validationErrors = append(
			validationErrors,
			"SERVER_READ_TIMEOUT_SEC must be greater than 0",
		)
	}

	if c.Server.WriteTimeoutSec <= 0 {
		validationErrors = append(
			validationErrors,
			"SERVER_WRITE_TIMEOUT_SEC must be greater than 0",
		)
	}

	if c.Server.IdleTimeoutSec <= 0 {
		validationErrors = append(
			validationErrors,
			"SERVER_IDLE_TIMEOUT_SEC must be greater than 0",
		)
	}

	if strings.TrimSpace(c.MongoDB.URI) == "" {
		validationErrors = append(
			validationErrors,
			"MONGODB_URI cannot be empty",
		)
	}

	if strings.TrimSpace(c.MongoDB.Database) == "" {
		validationErrors = append(
			validationErrors,
			"MONGODB_DATABASE cannot be empty",
		)
	}

	if c.Redis.DB < 0 {
		validationErrors = append(
			validationErrors,
			"REDIS_DB cannot be negative",
		)
	}

	if strings.TrimSpace(c.JWT.Secret) == "" {
		validationErrors = append(
			validationErrors,
			"JWT_SECRET cannot be empty",
		)
	}

	if len([]byte(c.JWT.Secret)) < 32 {
		validationErrors = append(
			validationErrors,
			"JWT_SECRET must be at least 32 bytes",
		)
	}

	if c.JWT.AccessTokenMinutes <= 0 {
		validationErrors = append(
			validationErrors,
			"JWT_ACCESS_TOKEN_MINUTES must be greater than 0",
		)
	}

	if c.Terraform.ExecutionTimeoutSec <= 0 {
		validationErrors = append(
			validationErrors,
			"TERRAFORM_EXECUTION_TIMEOUT_SEC must be greater than 0",
		)
	}

	if len(validationErrors) > 0 {
		return errors.New(strings.Join(validationErrors, "; "))
	}

	return nil
}

func getEnv(key string, fallback string) string {
	value, exists := os.LookupEnv(key)

	if !exists {
		return fallback
	}

	value = strings.TrimSpace(value)

	if value == "" {
		return fallback
	}

	return value
}

func getEnvInt(key string, fallback int) int {
	value, exists := os.LookupEnv(key)

	if !exists || strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}

	return parsed
}

func (c Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}