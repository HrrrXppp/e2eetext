package domain

import (
	"net/http"
	"reflect"
	"testing"
)

func TestOIDCProvider_AllowedCallbackMethods(t *testing.T) {
	tests := []struct {
		name         string
		responseMode string
		want         []string
	}{
		{
			name:         "no response_mode (Google-shaped) allows only GET",
			responseMode: "",
			want:         []string{http.MethodGet},
		},
		{
			name:         "response_mode=form_post (Apple) allows GET and POST",
			responseMode: ResponseModeFormPost,
			want:         []string{http.MethodGet, http.MethodPost},
		},
		{
			name:         "unrecognized response_mode allows only GET",
			responseMode: "query",
			want:         []string{http.MethodGet},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := OIDCProvider{ResponseMode: tt.responseMode}

			got := provider.AllowedCallbackMethods()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AllowedCallbackMethods() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
