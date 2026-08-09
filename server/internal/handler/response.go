package handler

import (
	"encoding/json"
	"errors"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	msg := err.Error()
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		msg = unwrapped.Error()
	}
	writeJSON(w, status, map[string]string{"error": msg})
}
