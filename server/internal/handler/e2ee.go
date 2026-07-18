package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ekhrunov/messenger/server/internal/node"
	"github.com/ekhrunov/messenger/server/internal/service"
)

type E2EEHandler struct {
	service *service.E2EEService
	node    node.Registry
}

func NewE2EEHandler(e2eeService *service.E2EEService, nodeRegistry node.Registry) *E2EEHandler {
	return &E2EEHandler{service: e2eeService, node: nodeRegistry}
}

type putIdentityKeyRequest struct {
	PublicKey json.RawMessage `json:"publicKey"`
}

func (h *E2EEHandler) PutIdentityKey(w http.ResponseWriter, r *http.Request) {
	tokenUser, ok := service.TokenUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	localID, err := h.node.LocalID(scopedIDFromPath(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req putIdentityKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.service.PutIdentityKey(r.Context(), localID, req.PublicKey, tokenUser); err != nil {
		if errors.Is(err, service.ErrE2EEAccessDenied) {
			writeError(w, http.StatusForbidden, err)
			return
		}
		if strings.Contains(err.Error(), "is required") || strings.Contains(err.Error(), "unsupported") || strings.Contains(err.Error(), "invalid") {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *E2EEHandler) GetIdentityKey(w http.ResponseWriter, r *http.Request) {
	if _, ok := service.TokenUserFromContext(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	localID, err := h.node.LocalID(scopedIDFromPath(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	item, err := h.service.GetIdentityKey(r.Context(), localID)
	if err != nil {
		if errors.Is(err, service.ErrIdentityKeyNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"publicKey": json.RawMessage(item.PublicKey),
	})
}

func (h *E2EEHandler) ListKeyWraps(w http.ResponseWriter, r *http.Request) {
	tokenUser, ok := service.TokenUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	chatID, err := h.node.LocalID(scopedIDFromPath(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	wraps, err := h.service.ListKeyWraps(r.Context(), chatID, tokenUser)
	if err != nil {
		if errors.Is(err, service.ErrE2EEAccessDenied) {
			writeError(w, http.StatusForbidden, err)
			return
		}
		if strings.Contains(err.Error(), "is required") {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	type apiWrap struct {
		KeyID     string          `json:"keyId"`
		Wrap      json.RawMessage `json:"wrap"`
		CreatedAt time.Time       `json:"createdAt"`
	}

	items := make([]apiWrap, 0, len(wraps))
	for _, item := range wraps {
		items = append(items, apiWrap{
			KeyID:     item.KeyID,
			Wrap:      item.Wrap,
			CreatedAt: item.CreatedAt.UTC(),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"wraps": items})
}
