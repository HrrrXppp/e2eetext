package node

import "testing"

func TestRegistry_ScopeAndLocalID(t *testing.T) {
	reg := Registry{ID: "99999999-9999-9999-9999-999999999999"}
	local := "11111111-1111-1111-1111-111111111111"

	scoped := reg.ScopeID(local)
	if scoped != "99999999-9999-9999-9999-999999999999/11111111-1111-1111-1111-111111111111" {
		t.Fatalf("ScopeID() = %q", scoped)
	}

	parsed, err := reg.LocalID(scoped)
	if err != nil {
		t.Fatalf("LocalID() error = %v", err)
	}
	if parsed != local {
		t.Fatalf("LocalID() = %q, want %q", parsed, local)
	}
}

func TestRegistry_LocalID_AcceptsBareUUID(t *testing.T) {
	reg := Registry{ID: "99999999-9999-9999-9999-999999999999"}
	local := "11111111-1111-1111-1111-111111111111"

	parsed, err := reg.LocalID(local)
	if err != nil {
		t.Fatalf("LocalID() error = %v", err)
	}
	if parsed != local {
		t.Fatalf("LocalID() = %q, want %q", parsed, local)
	}
}

func TestRegistry_ScopeID_Empty(t *testing.T) {
	reg := Registry{ID: "99999999-9999-9999-9999-999999999999"}
	if reg.ScopeID("") != "" {
		t.Fatalf("ScopeID(\"\") = %q, want empty", reg.ScopeID(""))
	}
	if reg.ScopeID("   ") != "" {
		t.Fatalf("ScopeID(\"   \") = %q, want empty", reg.ScopeID("   "))
	}
}

func TestRegistry_LocalID_InvalidScopedID(t *testing.T) {
	reg := Registry{ID: "99999999-9999-9999-9999-999999999999"}

	if _, err := reg.LocalID(""); err == nil {
		t.Fatal("LocalID(\"\") expected error")
	}
	if _, err := reg.LocalID("99999999-9999-9999-9999-999999999999/"); err == nil {
		t.Fatal("LocalID with empty local part expected error")
	}
}
