package version

import (
	"strings"
	"testing"
)

func TestVersion_IsSet(t *testing.T) {
	if strings.TrimSpace(Version) == "" {
		t.Fatal("Version is empty")
	}
}
