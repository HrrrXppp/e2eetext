package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ekhrunov/messenger/server/internal/node"
	"github.com/ekhrunov/messenger/server/internal/service"
)

type AuthHandler struct {
	service *service.AuthService
	node    node.Registry
}

func NewAuthHandler(authService *service.AuthService, nodeRegistry node.Registry) *AuthHandler {
	return &AuthHandler{service: authService, node: nodeRegistry}
}

func (h *AuthHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.service.ListProviders(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIProviderViews(h.node, providers))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	providerSlug := strings.ToLower(strings.TrimSpace(r.PathValue("provider")))
	if providerSlug == "" {
		writeError(w, http.StatusBadRequest, errors.New("provider is required"))
		return
	}

	if err := h.service.BeginLogin(w, r, providerSlug); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	providerSlug := strings.ToLower(strings.TrimSpace(r.PathValue("provider")))
	if providerSlug == "" {
		writeError(w, http.StatusBadRequest, errors.New("provider is required"))
		return
	}

	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		description := r.URL.Query().Get("error_description")
		if description != "" {
			writeError(w, http.StatusBadRequest, errors.New(description))
			return
		}
		writeError(w, http.StatusBadRequest, errors.New(errMsg))
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, errors.New("missing oauth code or state"))
		return
	}

	if err := h.service.CompleteLogin(w, r, providerSlug, code, state); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
}

type refreshTokenRequest struct {
	Provider     string `json:"provider"`
	RefreshToken string `json:"refreshToken"`
}

type refreshTokenResponse struct {
	IDToken      string `json:"idToken"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.RefreshTokens(r.Context(), service.RefreshTokenInput{
		Provider:     req.Provider,
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		if strings.Contains(err.Error(), "is required") {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	writeJSON(w, http.StatusOK, refreshTokenResponse{
		IDToken:      result.IDToken,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}
