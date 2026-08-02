package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// SupportedPrivateKeyJWTAlgorithm is the only private_key_jwt signing
// algorithm this server accepts. It is the single source of truth: config
// validation (below) fails closed on it early, and
// service.allowedPrivateKeyJWTAlgorithms is derived from it directly rather
// than repeating the "ES256" literal, so the two can't drift apart.
const SupportedPrivateKeyJWTAlgorithm = "ES256"

type OAuthCredential struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`

	// Optional private_key_jwt client-authentication fields (RFC 7523 /
	// OpenID Connect Core §9), used instead of ClientSecret whenever the
	// provider's client_secret_strategy is "private_key_jwt". Generic — not
	// specific to any one provider. For Apple: Issuer is the Team ID,
	// Subject is the Services ID (reuses ClientID), and Audience is the
	// provider's issuer URL (domain.OIDCProvider.Link).
	PrivateKeyJWTIssuer     string `json:"private_key_jwt_issuer,omitempty"`
	PrivateKeyJWTKeyID      string `json:"private_key_jwt_key_id,omitempty"`
	PrivateKeyJWTAlgorithm  string `json:"private_key_jwt_algorithm,omitempty"`
	PrivateKeyJWTPrivateKey string `json:"private_key_jwt_private_key,omitempty"`
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
		if strings.TrimSpace(credential.ClientSecret) == "" && strings.TrimSpace(credential.PrivateKeyJWTPrivateKey) == "" {
			return nil, fmt.Errorf("oauth_credentials.%s.client_secret (or private_key_jwt_private_key) is required", slug)
		}
		if strings.TrimSpace(credential.PrivateKeyJWTPrivateKey) != "" {
			// A signed client-assertion JWT is only correct with an explicit
			// issuer + key id; without them, minting silently produces a
			// JWT the IdP (Apple, mockoidc) rejects only at token exchange,
			// deep into the OAuth flow. Fail closed here instead.
			if strings.TrimSpace(credential.PrivateKeyJWTIssuer) == "" {
				return nil, fmt.Errorf("oauth_credentials.%s.private_key_jwt_issuer is required when private_key_jwt_private_key is set", slug)
			}
			if strings.TrimSpace(credential.PrivateKeyJWTKeyID) == "" {
				return nil, fmt.Errorf("oauth_credentials.%s.private_key_jwt_key_id is required when private_key_jwt_private_key is set", slug)
			}
			if credential.PrivateKeyJWTAlgorithm != "" && credential.PrivateKeyJWTAlgorithm != SupportedPrivateKeyJWTAlgorithm {
				return nil, fmt.Errorf("oauth_credentials.%s.private_key_jwt_algorithm %q is not supported (only %q)", slug, credential.PrivateKeyJWTAlgorithm, SupportedPrivateKeyJWTAlgorithm)
			}
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
