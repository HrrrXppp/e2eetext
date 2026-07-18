package e2ee

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const HybridIdentityPublicKeyBytes = 1216

func DecodeBase64URL(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("value is required")
	}

	padded := value
	switch len(value) % 4 {
	case 2:
		padded += "=="
	case 3:
		padded += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(padded)
	if err != nil {
		return nil, fmt.Errorf("invalid base64url")
	}

	return decoded, nil
}

func ValidateIdentityPublicKeyBytes(publicKey string) error {
	decoded, err := DecodeBase64URL(publicKey)
	if err != nil {
		return err
	}
	if len(decoded) != HybridIdentityPublicKeyBytes {
		return fmt.Errorf("identity public key has invalid length")
	}
	return nil
}
