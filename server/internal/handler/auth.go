package handler

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"net/url"
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

// Callback handles both GET (the default 302-redirect OIDC callback, e.g.
// Google) and POST (form_post response_mode, required by Apple whenever
// name/email scopes are requested) hitting the same provider callback URL,
// reading code/state/the one-time "user" JSON field from the query string
// or the POST form body respectively.
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	providerSlug := strings.ToLower(strings.TrimSpace(r.PathValue("provider")))
	if providerSlug == "" {
		writeError(w, http.StatusBadRequest, errors.New("provider is required"))
		return
	}

	values, err := callbackValues(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if errMsg := values.Get("error"); errMsg != "" {
		description := values.Get("error_description")
		if description != "" {
			writeError(w, http.StatusBadRequest, errors.New(description))
			return
		}
		writeError(w, http.StatusBadRequest, errors.New(errMsg))
		return
	}

	code := values.Get("code")
	state := values.Get("state")
	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, errors.New("missing oauth code or state"))
		return
	}

	oneTimeUser := values.Get("user")

	if err := h.service.CompleteLogin(w, r, providerSlug, code, state, oneTimeUser); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
}

// callbackValues reads code/state/user from wherever this provider actually
// put them: a form_post provider (Apple) POSTs them as an
// application/x-www-form-urlencoded body, so they must come from
// r.PostForm; the default 302-redirect flow (Google) appends them as query
// params on a GET, so they must come from r.URL.Query(). r.Form isn't used
// here because it would silently merge both sources instead of picking the
// one that matches how this request actually arrived.
//
// The signal for which source to use is the request's Content-Type, not
// its HTTP method: the method (GET vs POST) is just what each provider
// happens to pair with its response_mode, while Content-Type is what
// actually tells us whether there's an application/x-www-form-urlencoded
// body to parse.
func callbackValues(r *http.Request) (url.Values, error) {
	if isFormURLEncoded(r) {
		if err := r.ParseForm(); err != nil {
			return nil, err
		}
		return r.PostForm, nil
	}
	return r.URL.Query(), nil
}

// isFormURLEncoded reports whether r's body is
// application/x-www-form-urlencoded, ignoring any charset or other
// parameters on the media type.
func isFormURLEncoded(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}
	return mediaType == "application/x-www-form-urlencoded"
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
