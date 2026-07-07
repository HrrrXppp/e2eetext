package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ekhrunov/messenger/server/internal/node"
	"github.com/ekhrunov/messenger/server/internal/repository"
	"github.com/ekhrunov/messenger/server/internal/service"
)

type UserHandler struct {
	service *service.UserService
	node    node.Registry
}

func NewUserHandler(userService *service.UserService, nodeRegistry node.Registry) *UserHandler {
	return &UserHandler{service: userService, node: nodeRegistry}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	oidcProviderID, err := h.node.LocalID(strings.TrimSpace(r.URL.Query().Get("oidc_provider_id")))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	filter := repository.UserFilter{
		Subject:        strings.TrimSpace(r.URL.Query().Get("subject")),
		OIDCProviderID: oidcProviderID,
		Name:           strings.TrimSpace(r.URL.Query().Get("name")),
	}

	users, err := h.service.List(r.Context(), filter)
	if err != nil {
		if strings.Contains(err.Error(), "is required") {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIUsers(h.node, users))
}

func (h *UserHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	users, err := h.service.Search(r.Context(), query, h.node.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIUsers(h.node, users))
}

type createUserRequest struct {
	SkipProfile  bool   `json:"skip_profile"`
	KemPublicKey []byte `json:"kem_public_key"`
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	tokenUser, ok := service.TokenUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	req, err := decodeCreateUserRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	user, err := h.service.CreateFromToken(r.Context(), tokenUser, req.SkipProfile, req.KemPublicKey)
	if err != nil {
		if strings.Contains(err.Error(), "is required") || strings.Contains(err.Error(), "oidc provider not found") {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if errors.Is(err, repository.ErrDuplicateUser) {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAPIUser(h.node, user))
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	localID, err := h.node.LocalID(scopedIDFromPath(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	user, err := h.service.GetByID(r.Context(), localID)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIUser(h.node, user))
}

type updateUserRequest struct {
	Name string `json:"name"`
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	user, err := h.service.UpdateName(r.Context(), localID, req.Name, tokenUser)
	if err != nil {
		if errors.Is(err, service.ErrUserAccessDenied) || errors.Is(err, service.ErrChatAccessDenied) {
			writeError(w, http.StatusForbidden, err)
			return
		}
		if strings.Contains(err.Error(), "user not found") {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIUser(h.node, user))
}

func decodeCreateUserRequest(r *http.Request) (createUserRequest, error) {
	if r.Body == nil || r.Body == http.NoBody {
		return createUserRequest{}, nil
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req createUserRequest
	if err := decoder.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return createUserRequest{}, nil
		}
		return createUserRequest{}, err
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return createUserRequest{}, errors.New("request body must contain a single JSON object")
		}
		return createUserRequest{}, err
	}

	return req, nil
}
