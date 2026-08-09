package e2ee

import (
	"encoding/base64"
	"testing"
)

func TestValidateIdentityPublicKeyBytes(t *testing.T) {
	valid := make([]byte, HybridIdentityPublicKeyBytes)
	encoded := base64.RawURLEncoding.EncodeToString(valid)
	if err := ValidateIdentityPublicKeyBytes(encoded); err != nil {
		t.Fatalf("expected valid key, got %v", err)
	}

	if err := ValidateIdentityPublicKeyBytes(""); err == nil {
		t.Fatal("expected empty key to fail")
	}

	short := base64.RawURLEncoding.EncodeToString(make([]byte, HybridIdentityPublicKeyBytes-1))
	if err := ValidateIdentityPublicKeyBytes(short); err == nil {
		t.Fatal("expected short key to fail")
	}
}
