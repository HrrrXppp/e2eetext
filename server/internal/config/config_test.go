package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEnvFilesFindsRepoRoot(t *testing.T) {
	root := t.TempDir()
	envPath := filepath.Join(root, ".env")
	if err := os.WriteFile(envPath, []byte("SERVER_ADDR=:9090\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	nested := filepath.Join(root, "server", "cmd", "messenger")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	t.Chdir(nested)
	os.Unsetenv("SERVER_ADDR")

	loadEnvFiles()

	if got := os.Getenv("SERVER_ADDR"); got != ":9090" {
		t.Fatalf("SERVER_ADDR = %q, want :9090", got)
	}
}

func TestLoadReadsOAuthCredentials(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.json")
	configContents := `{
  "oauth_credentials": {
    "google": {
      "client_id": "test-client-id",
      "client_secret": "test-client-secret"
    }
  }
}`
	if err := os.WriteFile(configPath, []byte(configContents), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CONFIG_PATH", configPath)
	t.Setenv("DATABASE_URL", "postgres://messenger:messenger@localhost:5432/messenger?sslmode=disable")
	os.Unsetenv("SERVER_ADDR")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	credential, ok := cfg.OAuthForProvider("google")
	if !ok {
		t.Fatal("OAuthForProvider(google) = false, want true")
	}
	if credential.ClientID != "test-client-id" {
		t.Errorf("ClientID = %q, want test-client-id", credential.ClientID)
	}
	if credential.ClientSecret != "test-client-secret" {
		t.Errorf("ClientSecret = %q, want test-client-secret", credential.ClientSecret)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"oauth_credentials":{"google":{"client_id":"id","client_secret":"secret"}}}`), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CONFIG_PATH", configPath)
	t.Setenv("DATABASE_URL", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("Load() error = %v, want DATABASE_URL is required", err)
	}
}

func TestLoadRequiresConfigPath(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://messenger:messenger@localhost:5432/messenger?sslmode=disable")
	os.Unsetenv("CONFIG_PATH")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "CONFIG_PATH is required") {
		t.Fatalf("Load() error = %v, want CONFIG_PATH is required", err)
	}
}

func TestLoadRejectsMissingOAuthCredentials(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"oauth_credentials":{}}`), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CONFIG_PATH", configPath)
	t.Setenv("DATABASE_URL", "postgres://messenger:messenger@localhost:5432/messenger?sslmode=disable")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "oauth_credentials is required") {
		t.Fatalf("Load() error = %v, want oauth_credentials is required", err)
	}
}

func TestLoadRejectsMissingClientID(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.json")
	configContents := `{
  "oauth_credentials": {
    "google": {
      "client_secret": "test-client-secret"
    }
  }
}`
	if err := os.WriteFile(configPath, []byte(configContents), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CONFIG_PATH", configPath)
	t.Setenv("DATABASE_URL", "postgres://messenger:messenger@localhost:5432/messenger?sslmode=disable")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "client_id is required") {
		t.Fatalf("Load() error = %v, want client_id is required", err)
	}
}

func TestOAuthForProviderMissingSecret(t *testing.T) {
	_, err := normalizeOAuthCredentials(map[string]OAuthCredential{
		"google": {ClientID: "id-only"},
	})
	if err == nil || !strings.Contains(err.Error(), "client_secret is required") {
		t.Fatalf("normalizeOAuthCredentials() error = %v, want client_secret is required", err)
	}
}
