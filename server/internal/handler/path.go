package handler

import (
	"net/http"
	"strings"
)

func scopedIDFromPath(r *http.Request) string {
	nodeID := strings.TrimSpace(r.PathValue("nodeId"))
	localID := strings.TrimSpace(r.PathValue("localId"))
	if nodeID != "" && localID != "" {
		return nodeID + "/" + localID
	}

	return strings.TrimSpace(r.PathValue("id"))
}
