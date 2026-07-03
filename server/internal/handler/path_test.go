package handler

import (
	"net/http/httptest"
	"testing"
)

func TestScopedIDFromPath(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/user/99999999-9999-9999-9999-999999999999/11111111-1111-1111-1111-111111111111", nil)
	req.SetPathValue("nodeId", "99999999-9999-9999-9999-999999999999")
	req.SetPathValue("localId", "11111111-1111-1111-1111-111111111111")

	got := scopedIDFromPath(req)
	want := "99999999-9999-9999-9999-999999999999/11111111-1111-1111-1111-111111111111"
	if got != want {
		t.Fatalf("scopedIDFromPath() = %q, want %q", got, want)
	}
}

func TestScopedIDFromPath_FallsBackToSingleID(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/user/11111111-1111-1111-1111-111111111111", nil)
	req.SetPathValue("id", "11111111-1111-1111-1111-111111111111")

	got := scopedIDFromPath(req)
	if got != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("scopedIDFromPath() = %q", got)
	}
}
