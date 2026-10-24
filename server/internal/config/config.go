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
	RateLimits       RateLimits
	Metrics          MetricsConfig
	Sitemap          SitemapConfig
}

// SitemapConfig is the public host list and the pages a sitemap may name.
// Hosts[0] is the fallback when the request host is not in the list. A
// sitemap is only valid for the host that served it, so each host gets its
// own URL list built from Paths.
type SitemapConfig struct {
	Hosts []string
	Paths []string
}

// CanonicalHost is the host used when the request is not for one of Hosts.
func (s SitemapConfig) CanonicalHost() string {
	if len(s.Hosts) == 0 {
		return ""
	}
	return s.Hosts[0]
}

// DefaultSitemap is used when config.json omits the sitemap object, so an
// existing file keeps serving the four public names. The same values are
// written in server/config.example.json and maintenance/ec2/config/config.example.json.
func DefaultSitemap() SitemapConfig {
	return SitemapConfig{
		Hosts: []string{"e2eetext.com", "e2eetext.io", "e2eetext.net", "e2eetext.org"},
		Paths: []string{"/", "/privacy", "/terms", "/roadmap"},
	}
}

// WithDefaults fills a zero SitemapConfig from DefaultSitemap. A Config built
// in tests, without Load, has no sitemap section.
func (s SitemapConfig) WithDefaults() SitemapConfig {
	if len(s.Hosts) == 0 && len(s.Paths) == 0 {
		return DefaultSitemap()
	}
	return s
}

// MetricsConfig configures the optional CloudWatch publisher for this node's
// WebSocket connection count (#37). Publishing is enabled only when both
// Region and Namespace are set, so local dev and tests are unaffected.
type MetricsConfig struct {
	Region      string
	Namespace   string
	Environment string
}

// Enabled reports whether the CloudWatch publisher should run.
func (m MetricsConfig) Enabled() bool {
	return m.Region != "" && m.Namespace != ""
}

// RateLimitConfig describes one token bucket class: it refills at
// RequestsPerMinute and holds at most Burst tokens. A non-positive value in
// either field disables limiting for the class, matching
// middleware.RateLimitConfig's contract exactly (see toMiddlewareRateLimit
// in internal/app).
type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	Burst             int `json:"burst"`
}

// RateLimits groups the rate-limit classes the server applies per route
// group (see internal/app for the wiring). Auth guards the unauthenticated
// login/callback/refresh routes by IP; Search, Message, and Default guard
// authenticated routes by user, falling back to IP when unauthenticated; WS
// guards the WebSocket upgrade handshake by IP.
type RateLimits struct {
	Auth    RateLimitConfig
	Search  RateLimitConfig
	Message RateLimitConfig
	Default RateLimitConfig
	WS      RateLimitConfig
}

// defaultRateLimits are the conservative, hardcoded limits applied to any
// rate-limit class config.json omits entirely (see normalizeRateLimits).
// They're meant to be safe for both local dev and production out of the
// box; tune per environment in config.json if they're too tight or too
// loose for a given deployment.
func defaultRateLimits() RateLimits {
	return RateLimits{
		Auth:    RateLimitConfig{RequestsPerMinute: 30, Burst: 10},
		Search:  RateLimitConfig{RequestsPerMinute: 60, Burst: 20},
		Message: RateLimitConfig{RequestsPerMinute: 120, Burst: 30},
		Default: RateLimitConfig{RequestsPerMinute: 300, Burst: 60},
		WS:      RateLimitConfig{RequestsPerMinute: 30, Burst: 10},
	}
}

// fileRateLimits mirrors RateLimits for JSON decoding, but with pointer
// fields: that lets normalizeRateLimits tell a class config.json omits
// entirely (nil; use the default) apart from one explicitly present with a
// disabling non-positive value (use it as-is, per RateLimitConfig's
// contract) — a plain, non-pointer zero value can't distinguish the two.
type fileRateLimits struct {
	Auth    *RateLimitConfig `json:"auth"`
	Search  *RateLimitConfig `json:"search"`
	Message *RateLimitConfig `json:"message"`
	Default *RateLimitConfig `json:"default"`
	WS      *RateLimitConfig `json:"ws"`
}

// normalizeRateLimits fills in the built-in default for any class
// config.json omits, and leaves explicitly configured classes (including
// ones deliberately disabled with a non-positive value) untouched.
func normalizeRateLimits(limits fileRateLimits) RateLimits {
	defaults := defaultRateLimits()
	return RateLimits{
		Auth:    rateLimitOrDefault(limits.Auth, defaults.Auth),
		Search:  rateLimitOrDefault(limits.Search, defaults.Search),
		Message: rateLimitOrDefault(limits.Message, defaults.Message),
		Default: rateLimitOrDefault(limits.Default, defaults.Default),
		WS:      rateLimitOrDefault(limits.WS, defaults.WS),
	}
}

func rateLimitOrDefault(cfg *RateLimitConfig, def RateLimitConfig) RateLimitConfig {
	if cfg == nil {
		return def
	}
	return *cfg
}

type fileSitemap struct {
	Hosts []string `json:"hosts"`
	Paths []string `json:"paths"`
}

type fileConfig struct {
	OAuthCredentials map[string]OAuthCredential `json:"oauth_credentials"`
	RateLimits       fileRateLimits             `json:"rate_limits"`
	Sitemap          *fileSitemap               `json:"sitemap"`
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

	sitemap, err := normalizeSitemap(fileCfg.Sitemap)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:             addr,
		DatabaseURL:      databaseURL,
		OAuthCredentials: oauthCredentials,
		RateLimits:       normalizeRateLimits(fileCfg.RateLimits),
		Sitemap:          sitemap,
		Metrics: MetricsConfig{
			Region:      strings.TrimSpace(os.Getenv("AWS_REGION")),
			Namespace:   strings.TrimSpace(os.Getenv("METRICS_CLOUDWATCH_NAMESPACE")),
			Environment: strings.TrimSpace(os.Getenv("METRICS_ENVIRONMENT")),
		},
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

func normalizeSitemap(file *fileSitemap) (SitemapConfig, error) {
	if file == nil {
		return DefaultSitemap(), nil
	}
	if len(file.Hosts) == 0 {
		return SitemapConfig{}, fmt.Errorf("sitemap.hosts is required")
	}
	if len(file.Paths) == 0 {
		return SitemapConfig{}, fmt.Errorf("sitemap.paths is required")
	}

	hosts := make([]string, 0, len(file.Hosts))
	seenHosts := make(map[string]struct{}, len(file.Hosts))
	for _, raw := range file.Hosts {
		host, err := normalizeSitemapHost(raw)
		if err != nil {
			return SitemapConfig{}, err
		}
		if _, ok := seenHosts[host]; ok {
			return SitemapConfig{}, fmt.Errorf("sitemap.hosts contains duplicate %q", host)
		}
		seenHosts[host] = struct{}{}
		hosts = append(hosts, host)
	}

	paths := make([]string, 0, len(file.Paths))
	seenPaths := make(map[string]struct{}, len(file.Paths))
	for _, raw := range file.Paths {
		path, err := normalizeSitemapPath(raw)
		if err != nil {
			return SitemapConfig{}, err
		}
		if _, ok := seenPaths[path]; ok {
			return SitemapConfig{}, fmt.Errorf("sitemap.paths contains duplicate %q", path)
		}
		seenPaths[path] = struct{}{}
		paths = append(paths, path)
	}

	return SitemapConfig{Hosts: hosts, Paths: paths}, nil
}

func normalizeSitemapHost(raw string) (string, error) {
	host := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(raw)), ".")
	if host == "" || strings.ContainsAny(host, "/: \t") {
		return "", fmt.Errorf("sitemap.hosts entry %q must be a bare hostname", raw)
	}
	return host, nil
}

func normalizeSitemapPath(raw string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" || !strings.HasPrefix(path, "/") || strings.Contains(path, "://") || strings.ContainsAny(path, " \t") || strings.Contains(path, "..") {
		return "", fmt.Errorf("sitemap.paths entry %q must be an absolute path", raw)
	}
	return path, nil
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
