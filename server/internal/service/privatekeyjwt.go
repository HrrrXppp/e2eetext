package service

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	"github.com/ekhrunov/messenger/server/internal/config"
)

// allowedPrivateKeyJWTAlgorithms is the fail-closed set of signing
// algorithms this server will use to mint a private_key_jwt client
// assertion. jose.SignatureAlgorithm(algorithm) would otherwise accept any
// configured string; client authentication should stay on ECDSA
// (FIPS 186-5) rather than an accidental/misconfigured alg path. Derived
// from config.SupportedPrivateKeyJWTAlgorithm — the single source of truth
// shared with config validation — rather than repeating the "ES256" literal.
var allowedPrivateKeyJWTAlgorithms = map[string]bool{
	config.SupportedPrivateKeyJWTAlgorithm: true,
}

// mintPrivateKeyJWT signs a short-lived client-authentication JWT per the
// private_key_jwt method (RFC 7523 / OpenID Connect Core §9). This is a
// standard OAuth2/OIDC client authentication mechanism — not specific to any
// one identity provider — used in place of a static client_secret whenever a
// provider's client_secret_strategy is "private_key_jwt" (e.g. Apple, which
// requires it; Sign in with Apple has no static client secret at all).
func mintPrivateKeyJWT(
	issuer string,
	subject string,
	audience string,
	keyID string,
	algorithm string,
	privateKeyPEM string,
	ttl time.Duration,
) (string, error) {
	key, err := parseECPrivateKeyPEM(privateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}

	if algorithm == "" {
		algorithm = config.SupportedPrivateKeyJWTAlgorithm
	}
	if !allowedPrivateKeyJWTAlgorithms[algorithm] {
		return "", fmt.Errorf("private_key_jwt algorithm %q is not allowed", algorithm)
	}

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.SignatureAlgorithm(algorithm), Key: key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", keyID),
	)
	if err != nil {
		return "", fmt.Errorf("create signer: %w", err)
	}

	jti, err := randomToken(16)
	if err != nil {
		return "", fmt.Errorf("generate jti: %w", err)
	}

	now := time.Now()
	claims := jwt.Claims{
		Issuer:   issuer,
		Subject:  subject,
		Audience: jwt.Audience{audience},
		IssuedAt: jwt.NewNumericDate(now),
		Expiry:   jwt.NewNumericDate(now.Add(ttl)),
		ID:       jti,
	}

	signed, err := jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}

	return signed, nil
}

// parseECPrivateKeyPEM accepts either a traditional SEC1 "EC PRIVATE KEY"
// PEM block or a PKCS#8 "PRIVATE KEY" block wrapping an EC key.
func parseECPrivateKeyPEM(pemData string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM data")
	}

	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("not an EC private key: %w", err)
	}

	ecKey, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not an EC key")
	}

	return ecKey, nil
}
