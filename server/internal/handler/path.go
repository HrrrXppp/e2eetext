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

// scopedIDFromPathValues joins a node-id/local-id pair of path wildcards into a scoped ID.
func scopedIDFromPathValues(r *http.Request, nodeKey, localKey string) string {
	return strings.TrimSpace(r.PathValue(nodeKey)) + "/" + strings.TrimSpace(r.PathValue(localKey))
}
