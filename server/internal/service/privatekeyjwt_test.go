package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func generateTestECKeyPEM(t *testing.T) (string, *ecdsa.PrivateKey) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate EC key: %v", err)
	}

	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal EC key: %v", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	return string(pemBytes), key
}

func TestMintPrivateKeyJWT_ValidClaimsAndSignature(t *testing.T) {
	pemKey, key := generateTestECKeyPEM(t)

	signed, err := mintPrivateKeyJWT(
		"team-id-123",
		"services-id-456",
		"https://issuer.example.com",
		"test-key-id",
		"ES256",
		pemKey,
		5*time.Minute,
	)
	if err != nil {
		t.Fatalf("mintPrivateKeyJWT() error = %v", err)
	}

	parsed, err := jwt.ParseSigned(signed, []jose.SignatureAlgorithm{jose.ES256})
	if err != nil {
		t.Fatalf("ParseSigned() error = %v", err)
	}

	if len(parsed.Headers) != 1 {
		t.Fatalf("headers len = %d, want 1", len(parsed.Headers))
	}
	if parsed.Headers[0].KeyID != "test-key-id" {
		t.Errorf("kid = %q, want test-key-id", parsed.Headers[0].KeyID)
	}
	if parsed.Headers[0].Algorithm != string(jose.ES256) {
		t.Errorf("alg = %q, want ES256", parsed.Headers[0].Algorithm)
	}

	var claims jwt.Claims
	if err := parsed.Claims(&key.PublicKey, &claims); err != nil {
		t.Fatalf("verify signature: %v", err)
	}

	if claims.Issuer != "team-id-123" {
		t.Errorf("iss = %q, want team-id-123", claims.Issuer)
	}
	if claims.Subject != "services-id-456" {
		t.Errorf("sub = %q, want services-id-456", claims.Subject)
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != "https://issuer.example.com" {
		t.Errorf("aud = %v, want [https://issuer.example.com]", claims.Audience)
	}
	if claims.Expiry == nil || claims.IssuedAt == nil {
		t.Fatal("exp/iat claims must be set")
	}
	if claims.Expiry.Time().Before(claims.IssuedAt.Time()) {
		t.Errorf("exp must be after iat")
	}
	if claims.ID == "" {
		t.Error("jti claim must be set")
	}
}

func TestMintPrivateKeyJWT_JTIIsUniquePerAssertion(t *testing.T) {
	pemKey, key := generateTestECKeyPEM(t)

	signedA, err := mintPrivateKeyJWT("iss", "sub", "aud", "kid", "ES256", pemKey, time.Minute)
	if err != nil {
		t.Fatalf("mintPrivateKeyJWT() error = %v", err)
	}
	signedB, err := mintPrivateKeyJWT("iss", "sub", "aud", "kid", "ES256", pemKey, time.Minute)
	if err != nil {
		t.Fatalf("mintPrivateKeyJWT() error = %v", err)
	}

	jtiOf := func(signed string) string {
		parsed, err := jwt.ParseSigned(signed, []jose.SignatureAlgorithm{jose.ES256})
		if err != nil {
			t.Fatalf("ParseSigned() error = %v", err)
		}
		var claims jwt.Claims
		if err := parsed.Claims(&key.PublicKey, &claims); err != nil {
			t.Fatalf("verify signature: %v", err)
		}
		return claims.ID
	}

	jtiA, jtiB := jtiOf(signedA), jtiOf(signedB)
	if jtiA == "" || jtiB == "" {
		t.Fatal("jti claim must be set")
	}
	if jtiA == jtiB {
		t.Errorf("jti must be unique per assertion, got %q twice", jtiA)
	}
}

func TestMintPrivateKeyJWT_RejectsDisallowedAlgorithm(t *testing.T) {
	pemKey, _ := generateTestECKeyPEM(t)

	_, err := mintPrivateKeyJWT("iss", "sub", "aud", "kid", "HS256", pemKey, time.Minute)
	if err == nil {
		t.Fatal("mintPrivateKeyJWT() error = nil, want error for disallowed algorithm")
	}
}

func TestMintPrivateKeyJWT_DefaultsAlgorithmToES256(t *testing.T) {
	pemKey, key := generateTestECKeyPEM(t)

	signed, err := mintPrivateKeyJWT("iss", "sub", "aud", "kid", "", pemKey, time.Minute)
	if err != nil {
		t.Fatalf("mintPrivateKeyJWT() error = %v", err)
	}

	parsed, err := jwt.ParseSigned(signed, []jose.SignatureAlgorithm{jose.ES256})
	if err != nil {
		t.Fatalf("ParseSigned() error = %v", err)
	}

	var claims jwt.Claims
	if err := parsed.Claims(&key.PublicKey, &claims); err != nil {
		t.Fatalf("verify signature: %v", err)
	}
}

func TestMintPrivateKeyJWT_InvalidPEM(t *testing.T) {
	_, err := mintPrivateKeyJWT("iss", "sub", "aud", "kid", "ES256", "not a pem", time.Minute)
	if err == nil {
		t.Fatal("mintPrivateKeyJWT() error = nil, want error")
	}
}

func TestMintPrivateKeyJWT_RejectsRSAKey(t *testing.T) {
	// A well-formed PEM block that is not an EC key must be rejected.
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: []byte("not-a-real-key")}
	pemBytes := pem.EncodeToMemory(block)

	_, err := mintPrivateKeyJWT("iss", "sub", "aud", "kid", "ES256", string(pemBytes), time.Minute)
	if err == nil {
		t.Fatal("mintPrivateKeyJWT() error = nil, want error")
	}
}
