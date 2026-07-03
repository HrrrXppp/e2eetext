package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type OAuthCredential struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type Config struct {
	Addr             string
	DatabaseURL      string
	OAuthCredentials map[string]OAuthCredential
}

type fileConfig struct {
	OAuthCredentials map[string]OAuthCredential `json:"oauth_credentials"`
}

func Load() (Config, error) {
	loadEnvFiles()

	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	fileCfg, err := loadFileConfig()
	if err != nil {
		return Config{}, err
	}

	oauthCredentials, err := normalizeOAuthCredentials(fileCfg.OAuthCredentials)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:             addr,
		DatabaseURL:      databaseURL,
		OAuthCredentials: oauthCredentials,
	}, nil
}

func (c Config) OAuthForProvider(slug string) (OAuthCredential, bool) {
	credential, ok := c.OAuthCredentials[strings.ToLower(strings.TrimSpace(slug))]
	if !ok {
		return OAuthCredential{}, false
	}
	return credential, true
}

func (c Config) String() string {
	_, googleConfigured := c.OAuthForProvider("google")
	return fmt.Sprintf("addr=%s database=postgres google_oidc=%t", c.Addr, googleConfigured)
}

func loadFileConfig() (fileConfig, error) {
	configPath, err := configPath()
	if err != nil {
		return fileConfig{}, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fileConfig{}, fmt.Errorf("read config file: %w", err)
	}

	var cfg fileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fileConfig{}, fmt.Errorf("parse config file: %w", err)
	}

	return cfg, nil
}

func configPath() (string, error) {
	path := strings.TrimSpace(os.Getenv("CONFIG_PATH"))
	if path == "" {
		return "", fmt.Errorf("CONFIG_PATH is required")
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("config file not found: %s", path)
	}
	return path, nil
}

func normalizeOAuthCredentials(credentials map[string]OAuthCredential) (map[string]OAuthCredential, error) {
	if credentials == nil || len(credentials) == 0 {
		return nil, fmt.Errorf("oauth_credentials is required")
	}

	normalized := make(map[string]OAuthCredential, len(credentials))
	for slug, credential := range credentials {
		slug = strings.ToLower(strings.TrimSpace(slug))
		if slug == "" {
			return nil, fmt.Errorf("oauth_credentials provider slug is required")
		}
		if strings.TrimSpace(credential.ClientID) == "" {
			return nil, fmt.Errorf("oauth_credentials.%s.client_id is required", slug)
		}
		if strings.TrimSpace(credential.ClientSecret) == "" {
			return nil, fmt.Errorf("oauth_credentials.%s.client_secret is required", slug)
		}
		normalized[slug] = credential
	}

	return normalized, nil
}

func loadEnvFiles() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			_ = godotenv.Load(envPath)
			return
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}
