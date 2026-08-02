package domain

import (
	"net/http"
	"testing"
)

func TestOIDCProvider_AllowsCallbackMethod(t *testing.T) {
	tests := []struct {
		name         string
		responseMode string
		method       string
		want         bool
	}{
		{"GET always allowed (no response_mode)", "", http.MethodGet, true},
		{"POST rejected without form_post response_mode", "", http.MethodPost, false},
		{"POST allowed with form_post response_mode", ResponseModeFormPost, http.MethodPost, true},
		{"GET still allowed with form_post response_mode", ResponseModeFormPost, http.MethodGet, true},
		{"DELETE never allowed", ResponseModeFormPost, http.MethodDelete, false},
		{"GET allowed with unrecognized response_mode", "query", http.MethodGet, true},
		{"POST rejected with unrecognized response_mode", "query", http.MethodPost, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := OIDCProvider{ResponseMode: tt.responseMode}

			if got := provider.AllowsCallbackMethod(tt.method); got != tt.want {
				t.Errorf("AllowsCallbackMethod(%q) = %v, want %v", tt.method, got, tt.want)
			}
		})
	}
}
